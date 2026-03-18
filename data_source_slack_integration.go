package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*slackIntegrationDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*slackIntegrationDataSource)(nil)

type slackIntegrationDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSlackIntegrationDataSource() datasource.DataSource {
	return &slackIntegrationDataSource{}
}

func (d *slackIntegrationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *slackIntegrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_slack_integration"
}

func (d *slackIntegrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Slack integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The integration ID.",
			},
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the integration.",
			},
			"webhook_url": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The Slack webhook URL.",
			},
			"channel": schema.StringAttribute{
				Computed:    true,
				Description: "The Slack channel name.",
			},
			"axonops_url": schema.StringAttribute{
				Computed:    true,
				Description: "The AxonOps dashboard URL.",
			},
		},
	}
}

type slackIntegrationDataSourceData struct {
	ID          types.String `tfsdk:"id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	ClusterType types.String `tfsdk:"cluster_type"`
	Name        types.String `tfsdk:"name"`
	WebhookURL  types.String `tfsdk:"webhook_url"`
	Channel     types.String `tfsdk:"channel"`
	AxonopsURL  types.String `tfsdk:"axonops_url"`
}

func (d *slackIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data slackIntegrationDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	integrations, err := d.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "slack")
	if def == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Slack integration '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(def.ID)
	data.WebhookURL = types.StringValue(def.Params["url"])
	data.Channel = types.StringValue(def.Params["channel"])
	data.AxonopsURL = types.StringValue(def.Params["axondashUrl"])

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
