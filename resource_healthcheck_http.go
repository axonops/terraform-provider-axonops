package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*httpHealthcheckResource)(nil)
var _ resource.ResourceWithImportState = (*httpHealthcheckResource)(nil)

type httpHealthcheckResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewHTTPHealthcheckResource() resource.Resource {
	return &httpHealthcheckResource{}
}

func (r *httpHealthcheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *httpHealthcheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_healthcheck_http"
}

func (r *httpHealthcheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HTTP healthcheck configuration for a Kafka cluster.",
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
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The URL to check.",
			},
			"method": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("GET"),
				Description: "The HTTP method to use (GET, POST, etc.). Default: GET",
			},
			"headers": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				Description: "HTTP headers to include in the request.",
			},
			"body": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The request body for POST/PUT requests.",
			},
			"expected_status": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(200),
				Description: "The expected HTTP status code. Default: 200",
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

type httpHealthcheckResourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	URL                 types.String `tfsdk:"url"`
	Method              types.String `tfsdk:"method"`
	Headers             types.Map    `tfsdk:"headers"`
	Body                types.String `tfsdk:"body"`
	ExpectedStatus      types.Int64  `tfsdk:"expected_status"`
	Interval            types.String `tfsdk:"interval"`
	Timeout             types.String `tfsdk:"timeout"`
	Readonly            types.Bool   `tfsdk:"readonly"`
	SupportedAgentTypes types.List   `tfsdk:"supported_agent_types"`
}

func (r *httpHealthcheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data httpHealthcheckResourceData

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

	// Convert headers
	headers := make(map[string]string)
	diags = data.Headers.ElementsAs(ctx, &headers, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the new healthcheck
	newCheck := axonopsClient.HTTPHealthcheck{
		ID:                 newID,
		Name:               data.Name.ValueString(),
		URL:                data.URL.ValueString(),
		Method:             data.Method.ValueString(),
		Headers:            headers,
		Body:               data.Body.ValueString(),
		ExpectedStatus:     int(data.ExpectedStatus.ValueInt64()),
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
	existing.HTTPChecks = append(existing.HTTPChecks, newCheck)

	// Update all healthchecks
	err = r.client.UpdateHealthchecks(data.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create HTTP healthcheck, got error: %s", err))
		return
	}

	// Set the ID in state
	data.ID = types.StringValue(newID)

	tflog.Info(ctx, "Created HTTP healthcheck resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *httpHealthcheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data httpHealthcheckResourceData

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
	var found *axonopsClient.HTTPHealthcheck
	for _, c := range healthchecks.HTTPChecks {
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
	data.URL = types.StringValue(found.URL)
	data.Method = types.StringValue(found.Method)
	data.Body = types.StringValue(found.Body)
	data.ExpectedStatus = types.Int64Value(int64(found.ExpectedStatus))
	data.Interval = types.StringValue(found.Interval)
	data.Timeout = types.StringValue(found.Timeout)
	data.Readonly = types.BoolValue(found.Readonly)

	// Convert headers to map
	data.Headers, diags = types.MapValueFrom(ctx, types.StringType, found.Headers)
	resp.Diagnostics.Append(diags...)

	// Convert supported agent types to list
	data.SupportedAgentTypes, diags = types.ListValueFrom(ctx, types.StringType, found.SupportedAgentType)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *httpHealthcheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData httpHealthcheckResourceData
	var stateData httpHealthcheckResourceData

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

	// Convert headers
	headers := make(map[string]string)
	diags = planData.Headers.ElementsAs(ctx, &headers, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Find and update our healthcheck by name
	found := false
	for i, c := range existing.HTTPChecks {
		if c.Name == stateData.Name.ValueString() {
			existing.HTTPChecks[i] = axonopsClient.HTTPHealthcheck{
				ID:                 c.ID,
				Name:               planData.Name.ValueString(),
				URL:                planData.URL.ValueString(),
				Method:             planData.Method.ValueString(),
				Headers:            headers,
				Body:               planData.Body.ValueString(),
				ExpectedStatus:     int(planData.ExpectedStatus.ValueInt64()),
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
		resp.Diagnostics.AddError("Not Found", "HTTP healthcheck not found in cluster configuration")
		return
	}

	// Update all healthchecks
	err = r.client.UpdateHealthchecks(planData.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update HTTP healthcheck, got error: %s", err))
		return
	}

	// Keep the ID from state
	planData.ID = stateData.ID

	tflog.Info(ctx, "Updated HTTP healthcheck resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *httpHealthcheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data httpHealthcheckResourceData

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
	var updatedChecks []axonopsClient.HTTPHealthcheck
	for _, c := range existing.HTTPChecks {
		if c.Name != data.Name.ValueString() {
			updatedChecks = append(updatedChecks, c)
		}
	}
	existing.HTTPChecks = updatedChecks

	// Update all healthchecks (without our deleted one)
	err = r.client.UpdateHealthchecks(data.ClusterName.ValueString(), *existing)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete HTTP healthcheck, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted HTTP healthcheck resource")
}

// ImportState imports an existing HTTP healthcheck into Terraform state.
// Import ID format: cluster_name/healthcheck_name
func (r *httpHealthcheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_name/healthcheck_name, got: %s", req.ID),
		)
		return
	}

	clusterName := parts[0]
	healthcheckName := parts[1]

	// Get all healthchecks
	healthchecks, err := r.client.GetHealthchecks(clusterName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read healthchecks: %s", err),
		)
		return
	}

	// Find the HTTP healthcheck by name
	var found *axonopsClient.HTTPHealthcheck
	for _, c := range healthchecks.HTTPChecks {
		if c.Name == healthcheckName {
			found = &c
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("HTTP healthcheck %s not found in cluster %s", healthcheckName, clusterName),
		)
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), found.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("url"), found.URL)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("method"), found.Method)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("headers"), found.Headers)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("body"), found.Body)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("expected_status"), int64(found.ExpectedStatus))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("interval"), found.Interval)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timeout"), found.Timeout)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("readonly"), found.Readonly)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("supported_agent_types"), found.SupportedAgentType)...)

	tflog.Info(ctx, fmt.Sprintf("Imported HTTP healthcheck %s from cluster %s", healthcheckName, clusterName))
}
