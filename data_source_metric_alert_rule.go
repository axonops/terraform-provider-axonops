package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the alert rule.",
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
