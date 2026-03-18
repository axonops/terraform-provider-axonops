package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*opsgenieIntegrationDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*opsgenieIntegrationDataSource)(nil)

type opsgenieIntegrationDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewOpsGenieIntegrationDataSource() datasource.DataSource {
	return &opsgenieIntegrationDataSource{}
}

func (d *opsgenieIntegrationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *opsgenieIntegrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_opsgenie_integration"
}

func (d *opsgenieIntegrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an OpsGenie integration.",
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
			"opsgenie_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The OpsGenie API key.",
			},
		},
	}
}

type opsgenieIntegrationDataSourceData struct {
	ID          types.String `tfsdk:"id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	ClusterType types.String `tfsdk:"cluster_type"`
	Name        types.String `tfsdk:"name"`
	OpsgenieKey types.String `tfsdk:"opsgenie_key"`
}

func (d *opsgenieIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data opsgenieIntegrationDataSourceData
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

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "opsgenie")
	if def == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("OpsGenie integration '%s' not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(def.ID)
	data.OpsgenieKey = types.StringValue(def.Params["opsgenie_key"])

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
