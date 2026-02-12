package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*logCollectorDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*logCollectorDataSource)(nil)

type logCollectorDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewLogCollectorDataSource() datasource.DataSource {
	return &logCollectorDataSource{}
}

func (d *logCollectorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *logCollectorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logcollector"
}

func (d *logCollectorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a log collector configuration.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the log collector.",
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the log collector.",
			},
			"filename": schema.StringAttribute{
				Computed:    true,
				Description: "The log file path.",
			},
			"date_format": schema.StringAttribute{
				Computed:    true,
				Description: "The date format used in log entries.",
			},
			"info_regex": schema.StringAttribute{
				Computed:    true,
				Description: "Regex pattern for INFO level log entries.",
			},
			"warning_regex": schema.StringAttribute{
				Computed:    true,
				Description: "Regex pattern for WARNING level log entries.",
			},
			"error_regex": schema.StringAttribute{
				Computed:    true,
				Description: "Regex pattern for ERROR level log entries.",
			},
			"debug_regex": schema.StringAttribute{
				Computed:    true,
				Description: "Regex pattern for DEBUG level log entries.",
			},
			"supported_agent_types": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of agent types this collector supports.",
			},
			"error_alert_threshold": schema.Int64Attribute{
				Computed:    true,
				Description: "Threshold for error alerts.",
			},
		},
	}
}

type logCollectorDataSourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Name                types.String `tfsdk:"name"`
	UUID                types.String `tfsdk:"uuid"`
	Filename            types.String `tfsdk:"filename"`
	DateFormat          types.String `tfsdk:"date_format"`
	InfoRegex           types.String `tfsdk:"info_regex"`
	WarningRegex        types.String `tfsdk:"warning_regex"`
	ErrorRegex          types.String `tfsdk:"error_regex"`
	DebugRegex          types.String `tfsdk:"debug_regex"`
	SupportedAgentTypes types.List   `tfsdk:"supported_agent_types"`
	ErrorAlertThreshold types.Int64  `tfsdk:"error_alert_threshold"`
}

func (d *logCollectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data logCollectorDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	collectors, err := d.client.GetLogCollectors(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read log collectors: %s", err))
		return
	}

	var found *axonopsClient.LogCollectorConfig
	for _, c := range collectors {
		if c.Name == data.Name.ValueString() {
			found = &c
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Log collector %s not found", data.Name.ValueString()))
		return
	}

	data.UUID = types.StringValue(found.UUID)
	data.Filename = types.StringValue(found.Filename)
	data.DateFormat = types.StringValue(found.DateFormat)
	data.InfoRegex = types.StringValue(found.InfoRegex)
	data.WarningRegex = types.StringValue(found.WarningRegex)
	data.ErrorRegex = types.StringValue(found.ErrorRegex)
	data.DebugRegex = types.StringValue(found.DebugRegex)
	data.ErrorAlertThreshold = types.Int64Value(int64(found.ErrorAlertThreshold))

	data.SupportedAgentTypes, diags = types.ListValueFrom(ctx, types.StringType, found.SupportedAgentType)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
