package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "axonops-tf/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*metricAlertRuleResource)(nil)
var _ resource.ResourceWithImportState = (*metricAlertRuleResource)(nil)

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
				Computed:    true,
				Description: "The unique identifier for the alert rule (auto-generated).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the alert rule.",
			},
			"metric": schema.StringAttribute{
				Required:    true,
				Description: "The PromQL-style metric expression.",
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
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description of the alert rule.",
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
	Description   types.String  `tfsdk:"description"`
	Dc            types.List    `tfsdk:"dc"`
	Rack          types.List    `tfsdk:"rack"`
	HostId        types.List    `tfsdk:"host_id"`
	Scope         types.List    `tfsdk:"scope"`
	Keyspace      types.List    `tfsdk:"keyspace"`
	Percentile    types.List    `tfsdk:"percentile"`
	Consistency   types.List    `tfsdk:"consistency"`
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

func (r *metricAlertRuleResource) buildRule(data *metricAlertRuleResourceData, filters []axonopsClient.MetricAlertFilter) axonopsClient.MetricAlertRule {
	summary := fmt.Sprintf("%s is %s than threshold (current value: {{$value}})", data.Name.ValueString(), data.Operator.ValueString())

	return axonopsClient.MetricAlertRule{
		ID:            data.ID.ValueString(),
		Alert:         data.Name.ValueString(),
		For:           data.Duration.ValueString(),
		Operator:      data.Operator.ValueString(),
		WarningValue:  data.WarningValue.ValueFloat64(),
		CriticalValue: data.CriticalValue.ValueFloat64(),
		Expr:          data.Metric.ValueString(),
		Annotations: axonopsClient.MetricAlertAnnotations{
			Description: data.Description.ValueString(),
			Summary:     summary,
		},
		Filters: filters,
	}
}

func (r *metricAlertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data metricAlertRuleResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newID := uuid.New().String()
	data.ID = types.StringValue(newID)

	filters := r.buildFilters(ctx, &data)
	rule := r.buildRule(&data, filters)

	err := r.client.CreateOrUpdateAlertRule(data.ClusterType.ValueString(), data.ClusterName.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create alert rule: %s", err))
		return
	}

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
	data.Metric = types.StringValue(found.Expr)
	data.Operator = types.StringValue(found.Operator)
	data.WarningValue = types.Float64Value(found.WarningValue)
	data.CriticalValue = types.Float64Value(found.CriticalValue)
	data.Duration = types.StringValue(found.For)
	data.Description = types.StringValue(found.Annotations.Description)

	// Parse filters
	filterMap := map[string]*types.List{
		"dc":          &data.Dc,
		"rack":        &data.Rack,
		"host_id":     &data.HostId,
		"scope":       &data.Scope,
		"keyspace":    &data.Keyspace,
		"percentile":  &data.Percentile,
		"consistency": &data.Consistency,
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

	filters := r.buildFilters(ctx, &planData)
	rule := r.buildRule(&planData, filters)

	err := r.client.CreateOrUpdateAlertRule(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update alert rule: %s", err))
		return
	}

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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metric"), found.Expr)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("operator"), found.Operator)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("warning_value"), found.WarningValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("critical_value"), found.CriticalValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("duration"), found.For)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), found.Annotations.Description)...)

	// Parse filters into individual attributes
	filterMap := map[string]string{
		"dc":          "dc",
		"rack":        "rack",
		"host_id":     "host_id",
		"scope":       "scope",
		"keyspace":    "keyspace",
		"percentile":  "percentile",
		"consistency": "consistency",
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
