package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*connectorDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*connectorDataSource)(nil)

type connectorDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaConnectConnectorDataSource() datasource.DataSource {
	return &connectorDataSource{}
}

func (d *connectorDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *connectorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_connect_connector"
}

func (d *connectorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Kafka Connect connector.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"connect_cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka Connect cluster.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the connector.",
			},
			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The connector configuration.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the connector (source or sink).",
			},
		},
	}
}

type connectorDataSourceData struct {
	ClusterName        types.String            `tfsdk:"cluster_name"`
	ConnectClusterName types.String            `tfsdk:"connect_cluster_name"`
	Name               types.String            `tfsdk:"name"`
	Config             map[string]types.String `tfsdk:"config"`
	Type               types.String            `tfsdk:"type"`
}

func (d *connectorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data connectorDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetConnector(data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector: %s", err))
		return
	}

	if result == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Connector %s not found", data.Name.ValueString()))
		return
	}

	config := make(map[string]types.String)
	for key, value := range result.Config {
		if key == "name" {
			continue
		}
		config[key] = types.StringValue(value)
	}
	data.Config = config
	data.Type = types.StringValue(result.Type)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
