package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-kafka-tf/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*tcpHealthcheckResource)(nil)

type tcpHealthcheckResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewTCPHealthcheckResource() resource.Resource {
	return &tcpHealthcheckResource{}
}

func (r *tcpHealthcheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *tcpHealthcheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_healthcheck_tcp"
}

func (r *tcpHealthcheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TCP healthcheck configuration for a Kafka cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the healthcheck.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the healthcheck (auto-generated).",
			},
			"tcp": schema.StringAttribute{
				Required:    true,
				Description: "The TCP address to check (e.g., 0.0.0.0:9092).",
			},
			"interval": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1m"),
				Description: "The interval between checks (e.g., 1m, 30s). Default: 1m",
			},
			"timeout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1m"),
				Description: "The timeout for the check (e.g., 1m, 30s). Default: 1m",
			},
			"readonly": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the healthcheck is read-only. Default: false",
			},
			"supported_agent_types": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{types.StringValue("all")})),
				Description: "List of agent types this healthcheck applies to (e.g., all, broker, kraft-broker, kraft-controller, zookeeper).",
			},
		},
	}
}

type tcpHealthcheckResourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	TCP                 types.String `tfsdk:"tcp"`
	Interval            types.String `tfsdk:"interval"`
	Timeout             types.String `tfsdk:"timeout"`
	Readonly            types.Bool   `tfsdk:"readonly"`
	SupportedAgentTypes types.List   `tfsdk:"supported_agent_types"`
}

func (r *tcpHealthcheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tcpHealthcheckResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get existing healthchecks
	existing, err := r.client.GetHealthchecks(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get existing healthchecks, got error: %s", err))
		return
	}

	// Generate a new UUID for this healthcheck
	newID := uuid.New().String()

	// Convert supported agent types
	var supportedAgentTypes []string
	diags = data.SupportedAgentTypes.ElementsAs(ctx, &supportedAgentTypes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the new healthcheck
	newCheck := axonopsClient.TCPHealthcheck{
		ID:                 newID,
		Name:               data.Name.ValueString(),
		TCP:                data.TCP.ValueString(),
		Interval:           data.Interval.ValueString(),
		Timeout:            data.Timeout.ValueString(),
		Readonly:           data.Readonly.ValueBool(),
		SupportedAgentType: supportedAgentTypes,
		Integrations: axonopsClient.HealthcheckIntegrations{
			Type:            "",
			Routing:         nil,
			OverrideInfo:    false,
			OverrideWarning: false,
			OverrideError:   false,
		},
	}

	// Add to existing healthchecks
	existing.TCPChecks = append(existing.TCPChecks, newCheck)

	// Update all healthchecks
	err = r.client.UpdateHealthchecks(data.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create TCP healthcheck, got error: %s", err))
		return
	}

	// Set the ID in state
	data.ID = types.StringValue(newID)

	tflog.Info(ctx, "Created TCP healthcheck resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *tcpHealthcheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tcpHealthcheckResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get all healthchecks
	healthchecks, err := r.client.GetHealthchecks(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read healthchecks, got error: %s", err))
		return
	}

	// Find our healthcheck by name
	var found *axonopsClient.TCPHealthcheck
	for _, c := range healthchecks.TCPChecks {
		if c.Name == data.Name.ValueString() {
			found = &c
			break
		}
	}

	if found == nil {
		// Healthcheck was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with current values from API
	data.ID = types.StringValue(found.ID)
	data.TCP = types.StringValue(found.TCP)
	data.Interval = types.StringValue(found.Interval)
	data.Timeout = types.StringValue(found.Timeout)
	data.Readonly = types.BoolValue(found.Readonly)

	// Convert supported agent types to list
	data.SupportedAgentTypes, diags = types.ListValueFrom(ctx, types.StringType, found.SupportedAgentType)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *tcpHealthcheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData tcpHealthcheckResourceData
	var stateData tcpHealthcheckResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get existing healthchecks
	existing, err := r.client.GetHealthchecks(planData.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get existing healthchecks, got error: %s", err))
		return
	}

	// Convert supported agent types
	var supportedAgentTypes []string
	diags = planData.SupportedAgentTypes.ElementsAs(ctx, &supportedAgentTypes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Find and update our healthcheck by name
	found := false
	for i, c := range existing.TCPChecks {
		if c.Name == stateData.Name.ValueString() {
			existing.TCPChecks[i] = axonopsClient.TCPHealthcheck{
				ID:                 c.ID,
				Name:               planData.Name.ValueString(),
				TCP:                planData.TCP.ValueString(),
				Interval:           planData.Interval.ValueString(),
				Timeout:            planData.Timeout.ValueString(),
				Readonly:           planData.Readonly.ValueBool(),
				SupportedAgentType: supportedAgentTypes,
				Integrations:       c.Integrations,
			}
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Not Found", "TCP healthcheck not found in cluster configuration")
		return
	}

	// Update all healthchecks
	err = r.client.UpdateHealthchecks(planData.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update TCP healthcheck, got error: %s", err))
		return
	}

	// Keep the ID from state
	planData.ID = stateData.ID

	tflog.Info(ctx, "Updated TCP healthcheck resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *tcpHealthcheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tcpHealthcheckResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get existing healthchecks
	existing, err := r.client.GetHealthchecks(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get existing healthchecks, got error: %s", err))
		return
	}

	// Remove our healthcheck from the list
	var updatedChecks []axonopsClient.TCPHealthcheck
	for _, c := range existing.TCPChecks {
		if c.Name != data.Name.ValueString() {
			updatedChecks = append(updatedChecks, c)
		}
	}
	existing.TCPChecks = updatedChecks

	// Update all healthchecks (without our deleted one)
	err = r.client.UpdateHealthchecks(data.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete TCP healthcheck, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted TCP healthcheck resource")
}
