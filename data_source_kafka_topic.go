package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*topicDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*topicDataSource)(nil)

type topicDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaTopicDataSource() datasource.DataSource {
	return &topicDataSource{}
}

func (d *topicDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *topicDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_topic"
}

func (d *topicDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Kafka topic.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The topic name.",
			},
			"partitions": schema.Int32Attribute{
				Computed:    true,
				Description: "Number of partitions.",
			},
			"replication_factor": schema.Int32Attribute{
				Computed:    true,
				Description: "Replication factor.",
			},
			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Topic configuration (keys use underscores instead of dots).",
			},
		},
	}
}

type topicDataSourceData struct {
	ClusterName       types.String            `tfsdk:"cluster_name"`
	Name              types.String            `tfsdk:"name"`
	Partitions        types.Int32             `tfsdk:"partitions"`
	ReplicationFactor types.Int32             `tfsdk:"replication_factor"`
	Config            map[string]types.String `tfsdk:"config"`
}

func (d *topicDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data topicDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := d.client.GetTopic(data.Name.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read topic: %s", err))
		return
	}

	data.Partitions = types.Int32Value(topic.Partitions)
	data.ReplicationFactor = types.Int32Value(topic.ReplicationFactor)

	config := make(map[string]types.String)
	for _, c := range topic.Config {
		key := strings.ReplaceAll(c.Name, ".", "_")
		config[key] = types.StringValue(c.Value)
	}
	data.Config = config

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
