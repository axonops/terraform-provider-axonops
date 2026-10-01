package main

import (
	"context"
	"fmt"
	"sort"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*kafkaTopicsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*kafkaTopicsDataSource)(nil)

type kafkaTopicsDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaTopicsDataSource() datasource.DataSource {
	return &kafkaTopicsDataSource{}
}

func (d *kafkaTopicsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *kafkaTopicsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_topics"
}

func (d *kafkaTopicsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all topics in a Kafka cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"names": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Sorted list of topic names.",
			},
			"topics": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Topics in the cluster, sorted by name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The topic name.",
						},
						"is_internal": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this is a Kafka internal topic (for example `__consumer_offsets`).",
						},
						"partitions": schema.Int64Attribute{
							Computed:    true,
							Description: "The number of partitions.",
						},
						"replication_factor": schema.Int64Attribute{
							Computed:    true,
							Description: "The replication factor.",
						},
					},
				},
			},
		},
	}
}

type kafkaTopicsDataSourceData struct {
	ClusterName types.String           `tfsdk:"cluster_name"`
	Names       types.List             `tfsdk:"names"`
	Topics      []kafkaTopicsListEntry `tfsdk:"topics"`
}

type kafkaTopicsListEntry struct {
	Name              types.String `tfsdk:"name"`
	IsInternal        types.Bool   `tfsdk:"is_internal"`
	Partitions        types.Int64  `tfsdk:"partitions"`
	ReplicationFactor types.Int64  `tfsdk:"replication_factor"`
}

func (d *kafkaTopicsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data kafkaTopicsDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	topics, err := d.client.GetTopics(ctx, data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list Kafka topics: %s", err))
		return
	}

	sort.Slice(topics, func(i, j int) bool { return topics[i].Name < topics[j].Name })

	names := make([]string, 0, len(topics))
	entries := make([]kafkaTopicsListEntry, 0, len(topics))
	for _, t := range topics {
		names = append(names, t.Name)
		entries = append(entries, kafkaTopicsListEntry{
			Name:              types.StringValue(t.Name),
			IsInternal:        types.BoolValue(t.IsInternal),
			Partitions:        types.Int64Value(int64(t.Partitions)),
			ReplicationFactor: types.Int64Value(int64(t.ReplicationFactor)),
		})
	}
	data.Names = stringListValue(names)
	data.Topics = entries

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
