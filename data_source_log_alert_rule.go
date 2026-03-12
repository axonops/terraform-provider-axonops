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

var _ datasource.DataSource = (*logAlertRuleDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*logAlertRuleDataSource)(nil)

type logAlertRuleDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewLogAlertRuleDataSource() datasource.DataSource {
	return &logAlertRuleDataSource{}
}

func (d *logAlertRuleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *logAlertRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_alert_rule"
}

func (d *logAlertRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a log alert rule.",
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
				Description: "The unique identifier for the log alert rule.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the log alert rule.",
			},
			"content": schema.StringAttribute{
				Computed:    true,
				Description: "The log content/phrase to search for.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the log alert rule.",
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
				Description: "Duration/time window for log scraping.",
			},
			"level": schema.StringAttribute{
				Computed:    true,
				Description: "Log level filter.",
			},
			"log_type": schema.StringAttribute{
				Computed:    true,
				Description: "Log type filter.",
			},
			"source": schema.StringAttribute{
				Computed:    true,
				Description: "Log source path filter.",
			},
		},
	}
}

type logAlertRuleDataSourceData struct {
	ClusterName   types.String  `tfsdk:"cluster_name"`
	ClusterType   types.String  `tfsdk:"cluster_type"`
	ID            types.String  `tfsdk:"id"`
	Name          types.String  `tfsdk:"name"`
	Content       types.String  `tfsdk:"content"`
	Description   types.String  `tfsdk:"description"`
	Operator      types.String  `tfsdk:"operator"`
	WarningValue  types.Float64 `tfsdk:"warning_value"`
	CriticalValue types.Float64 `tfsdk:"critical_value"`
	Duration      types.String  `tfsdk:"duration"`
	Level         types.String  `tfsdk:"level"`
	LogType       types.String  `tfsdk:"log_type"`
	Source        types.String  `tfsdk:"source"`
}

func (d *logAlertRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data logAlertRuleDataSourceData

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
		if rule.ID == data.ID.ValueString() && strings.Contains(rule.Expr, "events{") {
			found = &rule
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Log alert rule %s not found", data.ID.ValueString()))
		return
	}

	content, level, source, logType := parseEventsExpr(found.Expr)

	data.Name = types.StringValue(found.Alert)
	data.Operator = types.StringValue(found.Operator)
	data.WarningValue = types.Float64Value(found.WarningValue)
	data.CriticalValue = types.Float64Value(found.CriticalValue)
	data.Duration = types.StringValue(found.For)
	data.Content = types.StringValue(content)
	data.Level = types.StringValue(level)
	data.LogType = types.StringValue(logType)
	data.Source = types.StringValue(source)
	data.Description = types.StringValue(found.Annotations.Description)

	tflog.Info(ctx, fmt.Sprintf("Read log alert rule %s", data.ID.ValueString()))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
