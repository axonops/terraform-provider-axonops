package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

// validRouteTypes lists the accepted values for the `type` attribute, derived
// from routeTypeMap so the two stay in sync.
var validRouteTypes = func() []string {
	types := make([]string, 0, len(routeTypeMap))
	for k := range routeTypeMap {
		types = append(types, k)
	}
	return types
}()

// validRouteSeverities lists the accepted values for the `severity` attribute.
var validRouteSeverities = []string{"info", "warning", "error"}

// validIntegrationTypes lists the accepted values for the `integration_type`
// attribute.
var validIntegrationTypes = []string{"email", "smtp", "pagerduty", "slack", "teams", "servicenow", "webhook", "opsgenie"}

// alertRouteID builds the composite identifier for an alert route from its
// six identity fields, matching the ImportState ID format.
func alertRouteID(clusterType, clusterName, routeType, severity, integrationType, integrationName string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", clusterType, clusterName, routeType, severity, integrationType, integrationName)
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
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier: cluster_type/cluster_name/type/severity/integration_type/integration_name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
				Validators:  []validator.String{clusterTypeValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the integration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of integration: email, smtp, pagerduty, slack, teams, servicenow, webhook, opsgenie.",
				Validators:  []validator.String{stringvalidator.OneOf(validIntegrationTypes...)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The route type: global, metrics, backups, servicechecks, nodes, commands, repairs, rollingrestart.",
				Validators:  []validator.String{stringvalidator.OneOf(validRouteTypes...)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"severity": schema.StringAttribute{
				Required:    true,
				Description: "The severity level: info, warning, error.",
				Validators:  []validator.String{stringvalidator.OneOf(validRouteSeverities...)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
	ID              types.String `tfsdk:"id"`
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
	integrations, err := r.client.GetIntegrations(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
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
		err = r.client.SetIntegrationOverride(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString(), apiRouteType, data.Severity.ValueString(), true)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set override: %s", err))
			return
		}
	}

	// Add the route
	err = r.client.AddIntegrationRoute(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString(), apiRouteType, data.Severity.ValueString(), integrationID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add route: %s", err))
		return
	}

	data.ID = types.StringValue(alertRouteID(
		data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.RouteType.ValueString(),
		data.Severity.ValueString(), data.IntegrationType.ValueString(), data.IntegrationName.ValueString(),
	))

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
	integrations, err := r.client.GetIntegrations(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
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

	data.ID = types.StringValue(alertRouteID(
		data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.RouteType.ValueString(),
		data.Severity.ValueString(), data.IntegrationType.ValueString(), data.IntegrationName.ValueString(),
	))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Update only ever runs for a change to enable_override: cluster_name,
// cluster_type, integration_name, integration_type, type, and severity are
// all RequiresReplace, so the route's identity never changes in place.
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

	apiRouteType, err := r.getAPIRouteType(planData.RouteType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", err.Error())
		return
	}

	integrations, err := r.client.GetIntegrations(ctx, planData.ClusterType.ValueString(), planData.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	integrationID, err := r.findIntegrationID(integrations, planData.IntegrationName.ValueString(), planData.IntegrationType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if planData.RouteType.ValueString() != "global" {
		if err := r.client.SetIntegrationOverride(ctx, planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), apiRouteType, planData.Severity.ValueString(), planData.EnableOverride.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set override: %s", err))
			return
		}
	}

	// The route itself is unaffected by enable_override; re-assert it in
	// case it was ever removed out-of-band.
	if err := r.client.AddIntegrationRoute(ctx, planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), apiRouteType, planData.Severity.ValueString(), integrationID); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add route: %s", err))
		return
	}

	planData.ID = types.StringValue(alertRouteID(
		planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), planData.RouteType.ValueString(),
		planData.Severity.ValueString(), planData.IntegrationType.ValueString(), planData.IntegrationName.ValueString(),
	))

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

	integrations, err := r.client.GetIntegrations(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	integrationID, err := r.findIntegrationID(integrations, data.IntegrationName.ValueString(), data.IntegrationType.ValueString())
	if err != nil {
		// Integration already gone, nothing to delete
		return
	}

	err = r.client.RemoveIntegrationRoute(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString(), apiRouteType, data.Severity.ValueString(), integrationID)
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
	integrations, err := r.client.GetIntegrations(ctx, clusterType, clusterName)
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

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), routeType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("severity"), severity)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integration_type"), integrationType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integration_name"), integrationName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enable_override"), enableOverride)...)

	tflog.Info(ctx, fmt.Sprintf("Imported alert route for %s/%s type=%s severity=%s", clusterType, clusterName, routeType, severity))
}
