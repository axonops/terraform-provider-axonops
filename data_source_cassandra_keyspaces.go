package main

import (
	"context"
	"fmt"
	"sort"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*cassandraKeyspacesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*cassandraKeyspacesDataSource)(nil)

type cassandraKeyspacesDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraKeyspacesDataSource() datasource.DataSource {
	return &cassandraKeyspacesDataSource{}
}

func (d *cassandraKeyspacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *cassandraKeyspacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_keyspaces"
}

func (d *cassandraKeyspacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the keyspaces of a Cassandra or DSE cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Cassandra or DSE cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Optional:    true,
				Description: "The cluster type: `cassandra` (default) or `dse`.",
				Validators:  []validator.String{stringvalidator.OneOf("cassandra", "dse")},
			},
			"include_system": schema.BoolAttribute{
				Optional:    true,
				Description: "Include system keyspaces (system, system_auth, system_schema, ...). Default: false.",
			},
			"names": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Sorted list of matching keyspace names.",
			},
			"keyspaces": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Matching keyspaces, sorted by name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The keyspace name.",
						},
						"replication_strategy": schema.StringAttribute{
							Computed:    true,
							Description: "The replication strategy class.",
						},
						"replication_factor": schema.Int64Attribute{
							Computed:    true,
							Description: "The replication factor reported by AxonOps.",
						},
						"replication_params": schema.StringAttribute{
							Computed:    true,
							Description: "The raw replication parameters (for example per-DC replication factors).",
						},
						"system": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this is a system keyspace.",
						},
						"tables": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "Sorted list of table names in the keyspace.",
						},
					},
				},
			},
		},
	}
}

type cassandraKeyspacesDataSourceData struct {
	ClusterName   types.String                 `tfsdk:"cluster_name"`
	ClusterType   types.String                 `tfsdk:"cluster_type"`
	IncludeSystem types.Bool                   `tfsdk:"include_system"`
	Names         types.List                   `tfsdk:"names"`
	Keyspaces     []cassandraKeyspaceListEntry `tfsdk:"keyspaces"`
}

type cassandraKeyspaceListEntry struct {
	Name                types.String `tfsdk:"name"`
	ReplicationStrategy types.String `tfsdk:"replication_strategy"`
	ReplicationFactor   types.Int64  `tfsdk:"replication_factor"`
	ReplicationParams   types.String `tfsdk:"replication_params"`
	System              types.Bool   `tfsdk:"system"`
	Tables              types.List   `tfsdk:"tables"`
}

func (d *cassandraKeyspacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data cassandraKeyspacesDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterType := "cassandra"
	if !data.ClusterType.IsNull() {
		clusterType = data.ClusterType.ValueString()
	}
	includeSystem := data.IncludeSystem.ValueBool()

	keyspaces, err := d.client.GetKeyspaces(ctx, clusterType, data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list keyspaces: %s", err))
		return
	}

	sort.Slice(keyspaces, func(i, j int) bool { return keyspaces[i].Name < keyspaces[j].Name })

	names := []string{}
	entries := []cassandraKeyspaceListEntry{}
	for _, ks := range keyspaces {
		if ks.System && !includeSystem {
			continue
		}
		tables := make([]string, 0, len(ks.Tables))
		for _, t := range ks.Tables {
			tables = append(tables, t.Name)
		}
		sort.Strings(tables)

		names = append(names, ks.Name)
		entries = append(entries, cassandraKeyspaceListEntry{
			Name:                types.StringValue(ks.Name),
			ReplicationStrategy: types.StringValue(ks.ReplicationStrategy),
			ReplicationFactor:   types.Int64Value(int64(ks.ReplicationFactor)),
			ReplicationParams:   types.StringValue(ks.ReplicationParams),
			System:              types.BoolValue(ks.System),
			Tables:              stringListValue(tables),
		})
	}
	data.Names = stringListValue(names)
	data.Keyspaces = entries

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
