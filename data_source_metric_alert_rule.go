package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = (*metricAlertRuleDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*metricAlertRuleDataSource)(nil)

type metricAlertRuleDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewMetricAlertRuleDataSource() datasource.DataSource {
	return &metricAlertRuleDataSource{}
}

func (d *metricAlertRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *metricAlertRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric_alert_rule"
}

func (d *metricAlertRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a metric alert rule.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier for the alert rule.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the alert rule.",
			},
			"metric": schema.StringAttribute{
				Computed:    true,
				Description: "The PromQL-style metric expression.",
			},
			"operator": schema.StringAttribute{
				Computed:    true,
				Description: "Comparison operator.",
			},
			"warning_value": schema.Float64Attribute{
				Computed:    true,
				Description: "Warning threshold value.",
			},
			"critical_value": schema.Float64Attribute{
				Computed:    true,
				Description: "Critical threshold value.",
			},
			"duration": schema.StringAttribute{
				Computed:    true,
				Description: "Duration before triggering.",
			},
			"dashboard": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the dashboard containing the chart for this alert.",
			},
			"chart": schema.StringAttribute{
				Computed:    true,
				Description: "The title of the chart (panel) within the dashboard for this alert.",
			},
			"correlation_id": schema.StringAttribute{
				Computed:    true,
				Description: "Correlation ID linking this alert to a dashboard widget.",
			},
			"annotations": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Alert rule annotations.",
				Attributes: map[string]schema.Attribute{
					"summary": schema.StringAttribute{
						Computed:    true,
						Description: "Summary template for the alert.",
					},
					"description": schema.StringAttribute{
						Computed:    true,
						Description: "Description of the alert rule.",
					},
					"widget_url": schema.StringAttribute{
						Computed:    true,
						Description: "URL to the associated dashboard widget.",
					},
				},
			},
			"integrations": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Integration routing configuration.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Computed:    true,
						Description: "Integration type.",
					},
					"routing": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						Description: "Routing keys or team identifiers.",
					},
					"override_info": schema.BoolAttribute{
						Computed:    true,
						Description: "Override info-level alerts.",
					},
					"override_warning": schema.BoolAttribute{
						Computed:    true,
						Description: "Override warning-level alerts.",
					},
					"override_error": schema.BoolAttribute{
						Computed:    true,
						Description: "Override error-level alerts.",
					},
				},
			},
			"dc": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Datacenter filters.",
			},
			"rack": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Rack filters.",
			},
			"host_id": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Host ID filters.",
			},
			"scope": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Scope filters.",
			},
			"keyspace": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Keyspace filters.",
			},
			"percentile": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Percentile filters.",
			},
			"consistency": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Consistency level filters.",
			},
			"topic": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Kafka topic filters.",
			},
			"group_id": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Kafka consumer group ID filters.",
			},
			"group_by": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Group by fields.",
			},
		},
	}
}

type metricAlertRuleDataSourceData struct {
	ClusterName   types.String  `tfsdk:"cluster_name"`
	ClusterType   types.String  `tfsdk:"cluster_type"`
	ID            types.String  `tfsdk:"id"`
	Name          types.String  `tfsdk:"name"`
	Metric        types.String  `tfsdk:"metric"`
	Operator      types.String  `tfsdk:"operator"`
	WarningValue  types.Float64 `tfsdk:"warning_value"`
	CriticalValue types.Float64 `tfsdk:"critical_value"`
	Duration      types.String  `tfsdk:"duration"`
	Dashboard     types.String  `tfsdk:"dashboard"`
	Chart         types.String  `tfsdk:"chart"`
	CorrelationId types.String  `tfsdk:"correlation_id"`
	Annotations   types.Object  `tfsdk:"annotations"`
	Integrations  types.Object  `tfsdk:"integrations"`
	Dc            types.List    `tfsdk:"dc"`
	Rack          types.List    `tfsdk:"rack"`
	HostId        types.List    `tfsdk:"host_id"`
	Scope         types.List    `tfsdk:"scope"`
	Keyspace      types.List    `tfsdk:"keyspace"`
	Percentile    types.List    `tfsdk:"percentile"`
	Consistency   types.List    `tfsdk:"consistency"`
	Topic         types.List    `tfsdk:"topic"`
	GroupId       types.List    `tfsdk:"group_id"`
	GroupBy       types.List    `tfsdk:"group_by"`
}

// duplicated from resource file per codebase convention (each file is self-contained)
var dsAnnotationsAttrTypes = map[string]attr.Type{
	"summary":     types.StringType,
	"description": types.StringType,
	"widget_url":  types.StringType,
}

var dsIntegrationsAttrTypes = map[string]attr.Type{
	"type":             types.StringType,
	"routing":          types.ListType{ElemType: types.StringType},
	"override_info":    types.BoolType,
	"override_warning": types.BoolType,
	"override_error":   types.BoolType,
}

func dsBuildAnnotationsObject(ctx context.Context, ann axonopsClient.MetricAlertAnnotations) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, dsAnnotationsAttrTypes, annotationsModel{
		Summary:     types.StringValue(ann.Summary),
		Description: types.StringValue(ann.Description),
		WidgetUrl:   types.StringValue(ann.WidgetUrl),
	})
}

func dsBuildIntegrationsObject(ctx context.Context, integ axonopsClient.MetricAlertIntegrations) (types.Object, diag.Diagnostics) {
	routing := integ.Routing
	if routing == nil {
		routing = []string{}
	}
	routingList, diags := types.ListValueFrom(ctx, types.StringType, routing)
	if diags.HasError() {
		return types.ObjectNull(dsIntegrationsAttrTypes), diags
	}
	obj, d := types.ObjectValueFrom(ctx, dsIntegrationsAttrTypes, integrationsModel{
		Type:            types.StringValue(integ.Type),
		Routing:         routingList,
		OverrideInfo:    types.BoolValue(integ.OverrideInfo),
		OverrideWarning: types.BoolValue(integ.OverrideWarning),
		OverrideError:   types.BoolValue(integ.OverrideError),
	})
	diags.Append(d...)
	return obj, diags
}

func (d *metricAlertRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data metricAlertRuleDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := d.client.GetAlertRules(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read alert rules: %s", err))
		return
	}

	var found *axonopsClient.MetricAlertRule
	for _, rule := range rules {
		if rule.ID == data.ID.ValueString() {
			found = &rule
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Alert rule %s not found", data.ID.ValueString()))
		return
	}

	data.Name = types.StringValue(found.Alert)
	// Strip trailing " operator value" from expression to get base metric
	data.Metric = types.StringValue(stripExprSuffix(found.Expr))
	data.Operator = types.StringValue(found.Operator)
	data.WarningValue = types.Float64Value(found.WarningValue)
	data.CriticalValue = types.Float64Value(found.CriticalValue)
	data.Duration = types.StringValue(found.For)
	data.CorrelationId = types.StringValue(found.CorrelationId)

	// Reverse-resolve correlation ID to dashboard/chart names
	if found.CorrelationId != "" {
		templates, err := d.client.GetDashboardTemplates(data.ClusterType.ValueString(), data.ClusterName.ValueString())
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Could not fetch dashboard templates: %s", err))
		} else {
			for _, dash := range templates.Dashboards {
				for _, panel := range dash.Panels {
					if panel.UUID == found.CorrelationId {
						data.Dashboard = types.StringValue(dash.Name)
						data.Chart = types.StringValue(panel.Title)
						break
					}
				}
				if !data.Dashboard.IsNull() {
					break
				}
			}
		}
	}
	if data.Dashboard.IsNull() {
		data.Dashboard = types.StringValue("")
	}
	if data.Chart.IsNull() {
		data.Chart = types.StringValue("")
	}

	data.Annotations, diags = dsBuildAnnotationsObject(ctx, found.Annotations)
	resp.Diagnostics.Append(diags...)
	data.Integrations, diags = dsBuildIntegrationsObject(ctx, found.Integrations)
	resp.Diagnostics.Append(diags...)

	// Parse filters
	filterMap := map[string]*types.List{
		"dc":          &data.Dc,
		"rack":        &data.Rack,
		"host_id":     &data.HostId,
		"scope":       &data.Scope,
		"keyspace":    &data.Keyspace,
		"percentile":  &data.Percentile,
		"consistency": &data.Consistency,
		"topic":       &data.Topic,
		"GroupID":     &data.GroupId,
		"groupBy":     &data.GroupBy,
	}

	// Set all filters to empty
	emptyList, _ := types.ListValueFrom(ctx, types.StringType, []string{})
	for _, v := range filterMap {
		*v = emptyList
	}

	// Set filters from API response
	for _, filter := range found.Filters {
		if target, ok := filterMap[filter.Name]; ok {
			*target, diags = types.ListValueFrom(ctx, types.StringType, filter.Value)
			resp.Diagnostics.Append(diags...)
		}
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
