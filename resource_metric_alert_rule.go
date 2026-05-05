package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*metricAlertRuleResource)(nil)
var _ resource.ResourceWithImportState = (*metricAlertRuleResource)(nil)

var annotationsAttrTypes = map[string]attr.Type{
	"summary":     types.StringType,
	"description": types.StringType,
	"widget_url":  types.StringType,
}

var integrationsAttrTypes = map[string]attr.Type{
	"type":             types.StringType,
	"routing":          types.ListType{ElemType: types.StringType},
	"override_info":    types.BoolType,
	"override_warning": types.BoolType,
	"override_error":   types.BoolType,
}

type annotationsModel struct {
	Summary     types.String `tfsdk:"summary"`
	Description types.String `tfsdk:"description"`
	WidgetUrl   types.String `tfsdk:"widget_url"`
}

type integrationsModel struct {
	Type            types.String `tfsdk:"type"`
	Routing         types.List   `tfsdk:"routing"`
	OverrideInfo    types.Bool   `tfsdk:"override_info"`
	OverrideWarning types.Bool   `tfsdk:"override_warning"`
	OverrideError   types.Bool   `tfsdk:"override_error"`
}

func buildAnnotationsObject(ctx context.Context, ann axonopsClient.MetricAlertAnnotations) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, annotationsAttrTypes, annotationsModel{
		Summary:     types.StringValue(ann.Summary),
		Description: types.StringValue(ann.Description),
		WidgetUrl:   types.StringValue(ann.WidgetUrl),
	})
}

func buildIntegrationsObject(ctx context.Context, integ axonopsClient.MetricAlertIntegrations) (types.Object, diag.Diagnostics) {
	routing := integ.Routing
	if routing == nil {
		routing = []string{}
	}
	routingList, diags := types.ListValueFrom(ctx, types.StringType, routing)
	if diags.HasError() {
		return types.ObjectNull(integrationsAttrTypes), diags
	}
	obj, d := types.ObjectValueFrom(ctx, integrationsAttrTypes, integrationsModel{
		Type:            types.StringValue(integ.Type),
		Routing:         routingList,
		OverrideInfo:    types.BoolValue(integ.OverrideInfo),
		OverrideWarning: types.BoolValue(integ.OverrideWarning),
		OverrideError:   types.BoolValue(integ.OverrideError),
	})
	diags.Append(d...)
	return obj, diags
}

type metricAlertRuleResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewMetricAlertRuleResource() resource.Resource {
	return &metricAlertRuleResource{}
}

func (r *metricAlertRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *metricAlertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric_alert_rule"
}

func (r *metricAlertRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptyList := listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{}))

	resp.Schema = schema.Schema{
		Description: "Manages a metric alert rule for a cluster.",
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
				Computed: true,
				Description: "Unique identifier for the alert rule. Derived deterministically from " +
					"org, cluster type, cluster name, rule name, and rule type — the same configuration " +
					"always produces the same ID, which makes Create idempotent across state loss and " +
					"transient API retries.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the alert rule.",
			},
			"metric": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The base metric expression. If omitted, auto-extracted from the chart's query.",
			},
			"operator": schema.StringAttribute{
				Required:    true,
				Description: "Comparison operator: >, >=, =, !=, <=, <",
			},
			"warning_value": schema.Float64Attribute{
				Required:    true,
				Description: "Warning threshold value.",
			},
			"critical_value": schema.Float64Attribute{
				Required:    true,
				Description: "Critical threshold value.",
			},
			"duration": schema.StringAttribute{
				Required:    true,
				Description: "Duration before triggering (e.g., 15m, 1h).",
			},
			"dashboard": schema.StringAttribute{
				Required:    true,
				Description: "The name of the dashboard containing the chart for this alert.",
			},
			"chart": schema.StringAttribute{
				Required:    true,
				Description: "The title of the chart (panel) within the dashboard for this alert.",
			},
			"correlation_id": schema.StringAttribute{
				Computed:    true,
				Description: "Correlation ID linking this alert to a dashboard widget (auto-resolved from dashboard/chart).",
			},
			"annotations": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Alert rule annotations (summary, description, widget URL).",
				Attributes: map[string]schema.Attribute{
					"summary": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Summary template for the alert. If omitted, auto-generated from name and operator.",
					},
					"description": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Description: "Description of the alert rule.",
					},
					"widget_url": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "URL to the associated dashboard widget. If omitted, auto-generated from dashboard/chart.",
					},
				},
			},
			"integrations": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Integration routing configuration for this alert rule.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Description: "Integration type (e.g., pagerduty, slack, email).",
					},
					"routing": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Computed:    true,
						Default:     emptyList,
						Description: "Routing keys or team identifiers.",
					},
					"override_info": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Override info-level alerts.",
					},
					"override_warning": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Override warning-level alerts.",
					},
					"override_error": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Override error-level alerts.",
					},
				},
			},
			"dc": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Datacenter filters.",
			},
			"rack": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Rack filters.",
			},
			"host_id": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Host ID filters.",
			},
			"scope": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Scope filters.",
			},
			"keyspace": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Keyspace filters.",
			},
			"percentile": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Percentile filters (e.g., 75thPercentile, 95thPercentile).",
			},
			"consistency": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Cassandra consistency level filters.",
			},
			"topic": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Kafka topic filters.",
			},
			"group_id": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Kafka consumer group ID filters.",
			},
			"group_by": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     emptyList,
				Description: "Group by fields (e.g., dc, host_id, rack, scope).",
			},
		},
	}
}

type metricAlertRuleResourceData struct {
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

func (r *metricAlertRuleResource) buildFilters(ctx context.Context, data *metricAlertRuleResourceData) []axonopsClient.MetricAlertFilter {
	var filters []axonopsClient.MetricAlertFilter

	filterMap := map[string]types.List{
		"dc":          data.Dc,
		"rack":        data.Rack,
		"host_id":     data.HostId,
		"scope":       data.Scope,
		"keyspace":    data.Keyspace,
		"percentile":  data.Percentile,
		"consistency": data.Consistency,
		"topic":       data.Topic,
		"GroupID":     data.GroupId,
		"groupBy":     data.GroupBy,
	}

	for name, list := range filterMap {
		var values []string
		list.ElementsAs(ctx, &values, false)
		if len(values) > 0 {
			filters = append(filters, axonopsClient.MetricAlertFilter{
				Name:  name,
				Value: values,
			})
		}
	}

	return filters
}

// stripExprSuffix removes the trailing " operator value" from a PromQL expression
// to recover the base metric. E.g. "foo{} >= 1" → "foo{}".
func stripExprSuffix(expr string) string {
	// Find the last two space-separated tokens and strip them
	// This matches the Ansible module pattern: re.sub(' [^ ]+ [^ ]+$', '', expr)
	parts := strings.Fields(expr)
	if len(parts) >= 3 {
		return strings.Join(parts[:len(parts)-2], " ")
	}
	return expr
}

// chartResolution holds the resolved dashboard/chart information.
type chartResolution struct {
	CorrelationId string
	WidgetUrl     string
	ChartQuery    string // raw query from the chart's first query, empty if none
}

// resolveDashboardChart resolves dashboard and chart names to UUIDs using the dashboard template API.
func (r *metricAlertRuleResource) resolveDashboardChart(clusterType, clusterName, dashboardName, chartTitle string) (*chartResolution, error) {
	templates, err := r.client.GetDashboardTemplates(clusterType, clusterName)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch dashboard templates: %s", err)
	}

	dashboardName = strings.ReplaceAll(dashboardName, "$$", "$")
	chartTitle = strings.ReplaceAll(chartTitle, "$$", "$")

	dash := axonopsClient.FindDashboardByName(templates, dashboardName)
	if dash == nil {
		return nil, fmt.Errorf("dashboard %q not found", dashboardName)
	}

	panel := axonopsClient.FindPanelByTitle(dash, chartTitle)
	if panel == nil {
		return nil, fmt.Errorf("chart %q not found in dashboard %q", chartTitle, dashboardName)
	}

	var chartQuery string
	if len(panel.Details.Queries) > 0 {
		chartQuery = panel.Details.Queries[0].Query
	}

	return &chartResolution{
		CorrelationId: panel.UUID,
		WidgetUrl: fmt.Sprintf("/%s/%s/%s/performance/%s?uuid=%s&time=30",
			r.client.OrgId(), clusterType, clusterName, dash.UUID, panel.UUID),
		ChartQuery: chartQuery,
	}, nil
}

// cleanChartQuery strips template variables (e.g. $dc, $rack) from a chart query
// to produce a base metric expression, following the Ansible module pattern.
func cleanChartQuery(query string) string {
	// Remove label filters referencing variables: key=~'$var', or key=~"$var",
	re := regexp.MustCompile(`\w+=~?['"]?\$\w*['"]?,?`)
	cleaned := re.ReplaceAllString(query, "")
	// Remove trailing comma before closing brace: , }
	cleaned = regexp.MustCompile(`, *}`).ReplaceAllString(cleaned, "}")
	// Replace ($groupBy) with (dc) as default
	cleaned = strings.ReplaceAll(cleaned, "($groupBy)", "(dc)")
	// Remove any remaining $variable references not caught above
	cleaned = regexp.MustCompile(`\$\w+`).ReplaceAllString(cleaned, "")
	// Clean up dangling commas in grouping parentheses: (dc, ) or (, dc) or (dc, , host_id)
	cleaned = regexp.MustCompile(`,\s*\)`).ReplaceAllString(cleaned, ")")
	cleaned = regexp.MustCompile(`\(\s*,`).ReplaceAllString(cleaned, "(")
	cleaned = regexp.MustCompile(`,\s*,`).ReplaceAllString(cleaned, ",")
	// Collapse multiple spaces
	cleaned = regexp.MustCompile(` +`).ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

// reverseLookupDashboardChart resolves a correlationId (chart UUID) back to dashboard/chart names.
func (r *metricAlertRuleResource) reverseLookupDashboardChart(clusterType, clusterName, correlationId string) (string, string, error) {
	templates, err := r.client.GetDashboardTemplates(clusterType, clusterName)
	if err != nil {
		return "", "", fmt.Errorf("unable to fetch dashboard templates: %s", err)
	}

	for _, dash := range templates.Dashboards {
		for _, panel := range dash.Panels {
			if panel.UUID == correlationId {
				return dash.Name, panel.Title, nil
			}
		}
	}

	return "", "", fmt.Errorf("could not find dashboard/chart for correlation ID %q", correlationId)
}

func (r *metricAlertRuleResource) buildRule(ctx context.Context, data *metricAlertRuleResourceData, filters []axonopsClient.MetricAlertFilter) axonopsClient.MetricAlertRule {
	// Extract annotations
	var ann annotationsModel
	if !data.Annotations.IsNull() && !data.Annotations.IsUnknown() {
		data.Annotations.As(ctx, &ann, basetypes.ObjectAsOptions{})
	}

	summary := ann.Summary.ValueString()
	if summary == "" {
		summary = fmt.Sprintf("%s is %s than threshold (current value: {{$value}})", data.Name.ValueString(), data.Operator.ValueString())
	}

	// Extract integrations
	var integ integrationsModel
	if !data.Integrations.IsNull() && !data.Integrations.IsUnknown() {
		data.Integrations.As(ctx, &integ, basetypes.ObjectAsOptions{})
	}

	var routing []string
	if !integ.Routing.IsNull() && !integ.Routing.IsUnknown() {
		integ.Routing.ElementsAs(ctx, &routing, false)
	}
	if routing == nil {
		routing = []string{}
	}

	return axonopsClient.MetricAlertRule{
		ID:            data.ID.ValueString(),
		CorrelationId: data.CorrelationId.ValueString(),
		WidgetTitle:   data.Chart.ValueString(),
		Alert:         data.Name.ValueString(),
		For:           data.Duration.ValueString(),
		Operator:      data.Operator.ValueString(),
		WarningValue:  data.WarningValue.ValueFloat64(),
		CriticalValue: data.CriticalValue.ValueFloat64(),
		Expr:          fmt.Sprintf("%s %s %s", data.Metric.ValueString(), data.Operator.ValueString(), strconv.FormatFloat(data.WarningValue.ValueFloat64(), 'f', -1, 64)),
		Annotations: axonopsClient.MetricAlertAnnotations{
			Description: ann.Description.ValueString(),
			Summary:     summary,
			WidgetUrl:   ann.WidgetUrl.ValueString(),
		},
		Filters: filters,
		Integrations: axonopsClient.MetricAlertIntegrations{
			Type:            integ.Type.ValueString(),
			Routing:         routing,
			OverrideInfo:    integ.OverrideInfo.ValueBool(),
			OverrideWarning: integ.OverrideWarning.ValueBool(),
			OverrideError:   integ.OverrideError.ValueBool(),
		},
	}
}

func (r *metricAlertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data metricAlertRuleResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(deterministicAlertRuleID(
		r.client.OrgId(),
		data.ClusterType.ValueString(),
		data.ClusterName.ValueString(),
		data.Name.ValueString(),
		"metric",
	))

	// Resolve dashboard/chart names to UUIDs
	resolved, err := r.resolveDashboardChart(
		data.ClusterType.ValueString(), data.ClusterName.ValueString(),
		data.Dashboard.ValueString(), data.Chart.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Dashboard Resolution Error", err.Error())
		return
	}
	data.CorrelationId = types.StringValue(resolved.CorrelationId)

	// Auto-extract metric from chart query if not provided
	if data.Metric.IsNull() || data.Metric.IsUnknown() || data.Metric.ValueString() == "" {
		if resolved.ChartQuery == "" {
			resp.Diagnostics.AddError("Metric Error", "metric is not set and the chart has no query to extract from")
			return
		}
		cleanedMetric := cleanChartQuery(resolved.ChartQuery)
		tflog.Debug(ctx, "Auto-extracted metric from chart query", map[string]interface{}{
			"raw_query":      resolved.ChartQuery,
			"cleaned_metric": cleanedMetric,
		})
		data.Metric = types.StringValue(cleanedMetric)
	}

	// Auto-set widget_url in annotations if not explicitly provided
	var ann annotationsModel
	if !data.Annotations.IsNull() && !data.Annotations.IsUnknown() {
		data.Annotations.As(ctx, &ann, basetypes.ObjectAsOptions{})
	}
	if ann.WidgetUrl.ValueString() == "" {
		ann.WidgetUrl = types.StringValue(resolved.WidgetUrl)
		data.Annotations, diags = types.ObjectValueFrom(ctx, annotationsAttrTypes, ann)
		resp.Diagnostics.Append(diags...)
	}

	filters := r.buildFilters(ctx, &data)
	rule := r.buildRule(ctx, &data, filters)
	tflog.Debug(ctx, "Built alert rule expression", map[string]interface{}{
		"expr": rule.Expr,
	})

	err = r.client.CreateOrUpdateAlertRule(data.ClusterType.ValueString(), data.ClusterName.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create alert rule: %s", err))
		return
	}

	// Write computed annotations/integrations back to state
	data.Annotations, diags = buildAnnotationsObject(ctx, rule.Annotations)
	resp.Diagnostics.Append(diags...)
	data.Integrations, diags = buildIntegrationsObject(ctx, rule.Integrations)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "Created metric alert rule resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *metricAlertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data metricAlertRuleResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := r.client.GetAlertRules(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read alert rules: %s", err))
		return
	}

	// Find rule by ID
	var found *axonopsClient.MetricAlertRule
	for _, rule := range rules {
		if rule.ID == data.ID.ValueString() {
			found = &rule
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
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
		dashName, chartName, err := r.reverseLookupDashboardChart(
			data.ClusterType.ValueString(), data.ClusterName.ValueString(), found.CorrelationId,
		)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Could not resolve dashboard/chart names from correlation ID: %s", err))
		} else {
			data.Dashboard = types.StringValue(dashName)
			data.Chart = types.StringValue(chartName)
		}
	}

	data.Annotations, diags = buildAnnotationsObject(ctx, found.Annotations)
	resp.Diagnostics.Append(diags...)
	data.Integrations, diags = buildIntegrationsObject(ctx, found.Integrations)
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

	// Reset all filters to empty
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

func (r *metricAlertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData metricAlertRuleResourceData
	var stateData metricAlertRuleResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keep the same ID
	planData.ID = stateData.ID

	// Resolve dashboard/chart names to UUIDs
	resolved, err := r.resolveDashboardChart(
		planData.ClusterType.ValueString(), planData.ClusterName.ValueString(),
		planData.Dashboard.ValueString(), planData.Chart.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Dashboard Resolution Error", err.Error())
		return
	}
	planData.CorrelationId = types.StringValue(resolved.CorrelationId)

	// Auto-extract metric from chart query if not provided
	if planData.Metric.IsNull() || planData.Metric.IsUnknown() || planData.Metric.ValueString() == "" {
		if resolved.ChartQuery == "" {
			resp.Diagnostics.AddError("Metric Error", "metric is not set and the chart has no query to extract from")
			return
		}
		planData.Metric = types.StringValue(cleanChartQuery(resolved.ChartQuery))
	}

	// Auto-set widget_url in annotations if not explicitly provided
	var ann annotationsModel
	if !planData.Annotations.IsNull() && !planData.Annotations.IsUnknown() {
		planData.Annotations.As(ctx, &ann, basetypes.ObjectAsOptions{})
	}
	if ann.WidgetUrl.ValueString() == "" {
		ann.WidgetUrl = types.StringValue(resolved.WidgetUrl)
		planData.Annotations, diags = types.ObjectValueFrom(ctx, annotationsAttrTypes, ann)
		resp.Diagnostics.Append(diags...)
	}

	filters := r.buildFilters(ctx, &planData)
	rule := r.buildRule(ctx, &planData, filters)

	err = r.client.CreateOrUpdateAlertRule(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update alert rule: %s", err))
		return
	}

	// Write computed annotations/integrations back to state
	planData.Annotations, diags = buildAnnotationsObject(ctx, rule.Annotations)
	resp.Diagnostics.Append(diags...)
	planData.Integrations, diags = buildIntegrationsObject(ctx, rule.Integrations)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "Updated metric alert rule resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *metricAlertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data metricAlertRuleResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAlertRule(data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete alert rule: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted metric alert rule resource")
}

// ImportState imports an existing alert rule.
// Import ID format: cluster_type/cluster_name/alert_id
func (r *metricAlertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/alert_id, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]
	alertID := parts[2]

	rules, err := r.client.GetAlertRules(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read alert rules: %s", err))
		return
	}

	var found *axonopsClient.MetricAlertRule
	for _, rule := range rules {
		if rule.ID == alertID {
			found = &rule
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Alert rule %s not found in cluster %s/%s", alertID, clusterType, clusterName))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), found.Alert)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metric"), stripExprSuffix(found.Expr))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("operator"), found.Operator)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("warning_value"), found.WarningValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("critical_value"), found.CriticalValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("duration"), found.For)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("correlation_id"), found.CorrelationId)...)

	// Reverse-resolve correlation ID to dashboard/chart names
	if found.CorrelationId != "" {
		dashName, chartName, err := r.reverseLookupDashboardChart(clusterType, clusterName, found.CorrelationId)
		if err != nil {
			resp.Diagnostics.AddWarning("Dashboard Resolution Warning",
				fmt.Sprintf("Could not resolve dashboard/chart names from correlation ID: %s", err))
		} else {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("dashboard"), dashName)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("chart"), chartName)...)
		}
	}

	annObj, diags := buildAnnotationsObject(ctx, found.Annotations)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("annotations"), annObj)...)

	if found.Integrations.Routing == nil {
		found.Integrations.Routing = []string{}
	}
	integObj, diags := buildIntegrationsObject(ctx, found.Integrations)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integrations"), integObj)...)

	// Parse filters into individual attributes
	filterMap := map[string]string{
		"dc":          "dc",
		"rack":        "rack",
		"host_id":     "host_id",
		"scope":       "scope",
		"keyspace":    "keyspace",
		"percentile":  "percentile",
		"consistency": "consistency",
		"topic":       "topic",
		"GroupID":     "group_id",
		"groupBy":     "group_by",
	}

	// Set empty defaults for all filter attributes
	for _, attr := range filterMap {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), []string{})...)
	}

	// Set filters from API response
	for _, filter := range found.Filters {
		if attr, ok := filterMap[filter.Name]; ok {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attr), filter.Value)...)
		}
	}

	tflog.Info(ctx, fmt.Sprintf("Imported metric alert rule %s from cluster %s/%s", alertID, clusterType, clusterName))
}
