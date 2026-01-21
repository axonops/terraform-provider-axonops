package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-kafka-tf/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*shellHealthcheckResource)(nil)

type shellHealthcheckResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewShellHealthcheckResource() resource.Resource {
	return &shellHealthcheckResource{}
}

func (r *shellHealthcheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *shellHealthcheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_healthcheck_shell"
}

func (r *shellHealthcheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a shell healthcheck configuration for a Kafka cluster.",
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
			"script": schema.StringAttribute{
				Required:    true,
				Description: "The script or command to execute (e.g., /usr/bin/ls, /path/to/script.sh).",
			},
			"shell": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The shell to use for executing the script (e.g., /bin/bash). Default: empty (uses default shell)",
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
		},
	}
}

type shellHealthcheckResourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	Script      types.String `tfsdk:"script"`
	Shell       types.String `tfsdk:"shell"`
	Interval    types.String `tfsdk:"interval"`
	Timeout     types.String `tfsdk:"timeout"`
	Readonly    types.Bool   `tfsdk:"readonly"`
}

func (r *shellHealthcheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data shellHealthcheckResourceData

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

	// Create the new healthcheck
	newCheck := axonopsClient.ShellHealthcheck{
		ID:       newID,
		Name:     data.Name.ValueString(),
		Script:   data.Script.ValueString(),
		Shell:    data.Shell.ValueString(),
		Interval: data.Interval.ValueString(),
		Timeout:  data.Timeout.ValueString(),
		Readonly: data.Readonly.ValueBool(),
		Integrations: axonopsClient.HealthcheckIntegrations{
			Type:            "",
			Routing:         nil,
			OverrideInfo:    false,
			OverrideWarning: false,
			OverrideError:   false,
		},
	}

	// Add to existing healthchecks
	existing.ShellChecks = append(existing.ShellChecks, newCheck)

	// Update all healthchecks
	err = r.client.UpdateHealthchecks(data.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create shell healthcheck, got error: %s", err))
		return
	}

	// Set the ID in state
	data.ID = types.StringValue(newID)

	tflog.Info(ctx, "Created shell healthcheck resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *shellHealthcheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data shellHealthcheckResourceData

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
	var found *axonopsClient.ShellHealthcheck
	for _, c := range healthchecks.ShellChecks {
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
	data.Script = types.StringValue(found.Script)
	data.Shell = types.StringValue(found.Shell)
	data.Interval = types.StringValue(found.Interval)
	data.Timeout = types.StringValue(found.Timeout)
	data.Readonly = types.BoolValue(found.Readonly)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *shellHealthcheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData shellHealthcheckResourceData
	var stateData shellHealthcheckResourceData

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

	// Find and update our healthcheck by name
	found := false
	for i, c := range existing.ShellChecks {
		if c.Name == stateData.Name.ValueString() {
			existing.ShellChecks[i] = axonopsClient.ShellHealthcheck{
				ID:           c.ID,
				Name:         planData.Name.ValueString(),
				Script:       planData.Script.ValueString(),
				Shell:        planData.Shell.ValueString(),
				Interval:     planData.Interval.ValueString(),
				Timeout:      planData.Timeout.ValueString(),
				Readonly:     planData.Readonly.ValueBool(),
				Integrations: c.Integrations,
			}
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Not Found", "Shell healthcheck not found in cluster configuration")
		return
	}

	// Update all healthchecks
	err = r.client.UpdateHealthchecks(planData.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update shell healthcheck, got error: %s", err))
		return
	}

	// Keep the ID from state
	planData.ID = stateData.ID

	tflog.Info(ctx, "Updated shell healthcheck resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *shellHealthcheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data shellHealthcheckResourceData

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
	var updatedChecks []axonopsClient.ShellHealthcheck
	for _, c := range existing.ShellChecks {
		if c.Name != data.Name.ValueString() {
			updatedChecks = append(updatedChecks, c)
		}
	}
	existing.ShellChecks = updatedChecks

	// Update all healthchecks (without our deleted one)
	err = r.client.UpdateHealthchecks(data.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete shell healthcheck, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted shell healthcheck resource")
}
