package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*kafkaConnectorsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*kafkaConnectorsDataSource)(nil)

type kafkaConnectorsDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaConnectorsDataSource() datasource.DataSource {
	return &kafkaConnectorsDataSource{}
}

func (d *kafkaConnectorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *kafkaConnectorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_connectors"
}

func (d *kafkaConnectorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the connectors in a Kafka Connect cluster attached to a Kafka cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"connect_cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka Connect cluster.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Only return connectors of this type. Valid values: `source`, `sink`.",
				Validators: []validator.String{
					stringvalidator.OneOf("source", "sink"),
				},
			},
			"names": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Sorted list of matching connector names.",
			},
			"connectors": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Matching connectors, sorted by name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The connector name.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The connector type (`source` or `sink`).",
						},
						"class": schema.StringAttribute{
							Computed:    true,
							Description: "The connector class (`connector.class`).",
						},
						"state": schema.StringAttribute{
							Computed:    true,
							Description: "The connector runtime state reported by Kafka Connect (for example `RUNNING`, `PAUSED`, `FAILED`). Empty when not reported.",
						},
						"tasks_max": schema.StringAttribute{
							Computed:    true,
							Description: "The configured `tasks.max`. Empty when not set.",
						},
					},
				},
			},
		},
	}
}

type kafkaConnectorsDataSourceData struct {
	ClusterName        types.String               `tfsdk:"cluster_name"`
	ConnectClusterName types.String               `tfsdk:"connect_cluster_name"`
	Type               types.String               `tfsdk:"type"`
	Names              types.List                 `tfsdk:"names"`
	Connectors         []kafkaConnectorsListEntry `tfsdk:"connectors"`
}

type kafkaConnectorsListEntry struct {
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Class    types.String `tfsdk:"class"`
	State    types.String `tfsdk:"state"`
	TasksMax types.String `tfsdk:"tasks_max"`
}

func (d *kafkaConnectorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data kafkaConnectorsDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectors, err := d.client.ListConnectors(ctx, data.ClusterName.ValueString(), data.ConnectClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list Kafka connectors: %s", err))
		return
	}

	names := []string{}
	entries := []kafkaConnectorsListEntry{}
	for _, name := range sortedKeys(connectors) {
		c := connectors[name]
		connType := c.Info.Type
		if connType == "" {
			connType = c.Status.Type
		}
		if !matchesFilter(data.Type, connType) {
			continue
		}
		names = append(names, name)
		entries = append(entries, kafkaConnectorsListEntry{
			Name:     types.StringValue(name),
			Type:     types.StringValue(connType),
			Class:    types.StringValue(c.Info.Config["connector.class"]),
			State:    types.StringValue(c.Status.Connector.State),
			TasksMax: types.StringValue(c.Info.Config["tasks.max"]),
		})
	}
	data.Names = stringListValue(names)
	data.Connectors = entries

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
