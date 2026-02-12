package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*alertRouteResource)(nil)
var _ resource.ResourceWithImportState = (*alertRouteResource)(nil)

// Route type mapping: Terraform name -> API URL-encoded name
var routeTypeMap = map[string]string{
	"global":         "Global",
	"metrics":        "Metrics",
	"backups":        "Backups",
	"servicechecks":  "Service%20Checks",
	"nodes":          "Nodes",
	"commands":       "Commands",
	"repairs":        "Repairs",
	"rollingrestart": "Rolling%20Restart",
}

type alertRouteResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewAlertRouteResource() resource.Resource {
	return &alertRouteResource{}
}

func (r *alertRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *alertRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_route"
}

func (r *alertRouteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an alert route to an integration (e.g., Slack, PagerDuty, email).",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
			},
			"integration_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the integration.",
			},
			"integration_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of integration: email, smtp, pagerduty, slack, teams, servicenow, webhook, opsgenie.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The route type: global, metrics, backups, servicechecks, nodes, commands, repairs, rollingrestart.",
			},
			"severity": schema.StringAttribute{
				Required:    true,
				Description: "The severity level: info, warning, error.",
			},
			"enable_override": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Enable override for non-global routes. Ignored for global routes. Default: true",
			},
		},
	}
}

type alertRouteResourceData struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	ClusterType     types.String `tfsdk:"cluster_type"`
	IntegrationName types.String `tfsdk:"integration_name"`
	IntegrationType types.String `tfsdk:"integration_type"`
	RouteType       types.String `tfsdk:"type"`
	Severity        types.String `tfsdk:"severity"`
	EnableOverride  types.Bool   `tfsdk:"enable_override"`
}

// findIntegrationID looks up the integration ID by name and type
func (r *alertRouteResource) findIntegrationID(integrations *axonopsClient.IntegrationsResponse, intName, intType string) (string, error) {
	for _, def := range integrations.Definitions {
		if strings.EqualFold(def.Type, intType) && strings.EqualFold(def.Params["name"], intName) {
			return def.ID, nil
		}
	}
	return "", fmt.Errorf("integration %s of type %s not found", intName, intType)
}

// getAPIRouteType converts the Terraform route type to the API URL-encoded type
func (r *alertRouteResource) getAPIRouteType(tfType string) (string, error) {
	apiType, ok := routeTypeMap[tfType]
	if !ok {
		return "", fmt.Errorf("unknown route type: %s", tfType)
	}
	return apiType, nil
}

func (r *alertRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data alertRouteResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRouteType, err := r.getAPIRouteType(data.RouteType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", err.Error())
		return
	}

	// Get integrations to find the integration ID
	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	integrationID, err := r.findIntegrationID(integrations, data.IntegrationName.ValueString(), data.IntegrationType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Set override if non-global and enabled
	if data.RouteType.ValueString() != "global" && data.EnableOverride.ValueBool() {
		err = r.client.SetIntegrationOverride(data.ClusterType.ValueString(), data.ClusterName.ValueString(), apiRouteType, data.Severity.ValueString(), true)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set override: %s", err))
			return
		}
	}

	// Add the route
	err = r.client.AddIntegrationRoute(data.ClusterType.ValueString(), data.ClusterName.ValueString(), apiRouteType, data.Severity.ValueString(), integrationID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add route: %s", err))
		return
	}

	tflog.Info(ctx, "Created alert route resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *alertRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data alertRouteResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRouteType, err := r.getAPIRouteType(data.RouteType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", err.Error())
		return
	}

	// Get integrations
	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	integrationID, err := r.findIntegrationID(integrations, data.IntegrationName.ValueString(), data.IntegrationType.ValueString())
	if err != nil {
		// Integration no longer exists
		resp.State.RemoveResource(ctx)
		return
	}

	// Check if route exists
	routeFound := false
	// Decode the API route type for comparison (URL-decode %20 to space)
	decodedAPIRouteType := strings.ReplaceAll(apiRouteType, "%20", " ")
	for _, routing := range integrations.Routings {
		if routing.Type == decodedAPIRouteType {
			for _, route := range routing.Routing {
				if route.ID == integrationID && strings.EqualFold(route.Severity, data.Severity.ValueString()) {
					routeFound = true
					break
				}
			}
			// Read override state
			if data.RouteType.ValueString() != "global" {
				switch strings.ToLower(data.Severity.ValueString()) {
				case "info":
					data.EnableOverride = types.BoolValue(routing.OverrideInfo)
				case "warning":
					data.EnableOverride = types.BoolValue(routing.OverrideWarning)
				case "error":
					data.EnableOverride = types.BoolValue(routing.OverrideError)
				}
			}
			break
		}
	}

	if !routeFound {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *alertRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData alertRouteResourceData
	var stateData alertRouteResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove old route
	oldAPIRouteType, err := r.getAPIRouteType(stateData.RouteType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", err.Error())
		return
	}

	integrations, err := r.client.GetIntegrations(stateData.ClusterType.ValueString(), stateData.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	oldIntegrationID, err := r.findIntegrationID(integrations, stateData.IntegrationName.ValueString(), stateData.IntegrationType.ValueString())
	if err == nil {
		_ = r.client.RemoveIntegrationRoute(stateData.ClusterType.ValueString(), stateData.ClusterName.ValueString(), oldAPIRouteType, stateData.Severity.ValueString(), oldIntegrationID)
	}

	// Add new route
	newAPIRouteType, err := r.getAPIRouteType(planData.RouteType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", err.Error())
		return
	}

	// Re-fetch integrations if cluster changed
	if planData.ClusterName.ValueString() != stateData.ClusterName.ValueString() || planData.ClusterType.ValueString() != stateData.ClusterType.ValueString() {
		integrations, err = r.client.GetIntegrations(planData.ClusterType.ValueString(), planData.ClusterName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
			return
		}
	}

	newIntegrationID, err := r.findIntegrationID(integrations, planData.IntegrationName.ValueString(), planData.IntegrationType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Set override
	if planData.RouteType.ValueString() != "global" && planData.EnableOverride.ValueBool() {
		err = r.client.SetIntegrationOverride(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), newAPIRouteType, planData.Severity.ValueString(), true)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set override: %s", err))
			return
		}
	}

	err = r.client.AddIntegrationRoute(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), newAPIRouteType, planData.Severity.ValueString(), newIntegrationID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add route: %s", err))
		return
	}

	tflog.Info(ctx, "Updated alert route resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *alertRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data alertRouteResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRouteType, err := r.getAPIRouteType(data.RouteType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", err.Error())
		return
	}

	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	integrationID, err := r.findIntegrationID(integrations, data.IntegrationName.ValueString(), data.IntegrationType.ValueString())
	if err != nil {
		// Integration already gone, nothing to delete
		return
	}

	err = r.client.RemoveIntegrationRoute(data.ClusterType.ValueString(), data.ClusterName.ValueString(), apiRouteType, data.Severity.ValueString(), integrationID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove route: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted alert route resource")
}

// ImportState imports an existing alert route.
// Import ID format: cluster_type/cluster_name/type/severity/integration_type/integration_name
func (r *alertRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 6 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/type/severity/integration_type/integration_name, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]
	routeType := parts[2]
	severity := parts[3]
	integrationType := parts[4]
	integrationName := parts[5]

	// Validate route type
	_, err := r.getAPIRouteType(routeType)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", err.Error())
		return
	}

	// Verify the integration exists
	integrations, err := r.client.GetIntegrations(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	_, err = r.findIntegrationID(integrations, integrationName, integrationType)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", err.Error())
		return
	}

	// Read override state
	enableOverride := false
	if routeType != "global" {
		apiRouteType, _ := r.getAPIRouteType(routeType)
		decodedAPIRouteType := strings.ReplaceAll(apiRouteType, "%20", " ")
		for _, routing := range integrations.Routings {
			if routing.Type == decodedAPIRouteType {
				switch strings.ToLower(severity) {
				case "info":
					enableOverride = routing.OverrideInfo
				case "warning":
					enableOverride = routing.OverrideWarning
				case "error":
					enableOverride = routing.OverrideError
				}
				break
			}
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), routeType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("severity"), severity)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integration_type"), integrationType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integration_name"), integrationName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enable_override"), enableOverride)...)

	tflog.Info(ctx, fmt.Sprintf("Imported alert route for %s/%s type=%s severity=%s", clusterType, clusterName, routeType, severity))
}
