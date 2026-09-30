package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = (*alertRouteDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*alertRouteDataSource)(nil)

type alertRouteDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewAlertRouteDataSource() datasource.DataSource {
	return &alertRouteDataSource{}
}

func (d *alertRouteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *alertRouteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_route"
}

func (d *alertRouteDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an existing alert route to an integration (e.g., Slack, PagerDuty, email).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier: cluster_type/cluster_name/type/severity/integration_type/integration_name.",
			},
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
				Computed:    true,
				Description: "Whether override is enabled for this route. Always false for global routes.",
			},
		},
	}
}

type alertRouteDataSourceData struct {
	ID              types.String `tfsdk:"id"`
	ClusterName     types.String `tfsdk:"cluster_name"`
	ClusterType     types.String `tfsdk:"cluster_type"`
	IntegrationName types.String `tfsdk:"integration_name"`
	IntegrationType types.String `tfsdk:"integration_type"`
	RouteType       types.String `tfsdk:"type"`
	Severity        types.String `tfsdk:"severity"`
	EnableOverride  types.Bool   `tfsdk:"enable_override"`
}

// findAlertRouteIntegrationID looks up the integration ID by name and type,
// mirroring alertRouteResource.findIntegrationID.
func findAlertRouteIntegrationID(integrations *axonopsClient.IntegrationsResponse, intName, intType string) (string, error) {
	for _, def := range integrations.Definitions {
		if strings.EqualFold(def.Type, intType) && strings.EqualFold(def.Params["name"], intName) {
			return def.ID, nil
		}
	}
	return "", fmt.Errorf("integration %s of type %s not found", intName, intType)
}

func (d *alertRouteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data alertRouteDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRouteType, ok := routeTypeMap[data.RouteType.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("unknown route type: %s", data.RouteType.ValueString()))
		return
	}

	integrations, err := d.client.GetIntegrations(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	integrationID, err := findAlertRouteIntegrationID(integrations, data.IntegrationName.ValueString(), data.IntegrationType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Not Found", err.Error())
		return
	}

	decodedAPIRouteType := strings.ReplaceAll(apiRouteType, "%20", " ")
	routeFound := false
	enableOverride := false
	for _, routing := range integrations.Routings {
		if routing.Type != decodedAPIRouteType {
			continue
		}
		for _, route := range routing.Routing {
			if route.ID == integrationID && strings.EqualFold(route.Severity, data.Severity.ValueString()) {
				routeFound = true
				break
			}
		}
		if data.RouteType.ValueString() != "global" {
			switch strings.ToLower(data.Severity.ValueString()) {
			case "info":
				enableOverride = routing.OverrideInfo
			case "warning":
				enableOverride = routing.OverrideWarning
			case "error":
				enableOverride = routing.OverrideError
			}
		}
		break
	}

	if !routeFound {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf(
			"Alert route not found for cluster %s/%s type=%s severity=%s integration=%s/%s",
			data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.RouteType.ValueString(),
			data.Severity.ValueString(), data.IntegrationType.ValueString(), data.IntegrationName.ValueString(),
		))
		return
	}

	data.EnableOverride = types.BoolValue(enableOverride)
	data.ID = types.StringValue(alertRouteID(
		data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.RouteType.ValueString(),
		data.Severity.ValueString(), data.IntegrationType.ValueString(), data.IntegrationName.ValueString(),
	))

	tflog.Info(ctx, fmt.Sprintf("Read alert route for %s/%s type=%s severity=%s",
		data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.RouteType.ValueString(), data.Severity.ValueString()))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
