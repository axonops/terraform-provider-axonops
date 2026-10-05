package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*clusterDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*clusterDataSource)(nil)

type clusterDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewClusterDataSource() datasource.DataSource {
	return &clusterDataSource{}
}

func (d *clusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *clusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *clusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := clusterSummaryAttributes()
	attrs["name"] = schema.StringAttribute{
		Required:    true,
		Description: "The cluster name.",
	}
	attrs["type"] = schema.StringAttribute{
		Optional: true,
		Computed: true,
		Description: "The cluster type (cassandra, kafka, or dse). Required only when clusters of different types " +
			"share the same name.",
		Validators: []validator.String{clusterTypeValidator()},
	}
	attrs["node_count"] = schema.Int64Attribute{
		Computed:    true,
		Description: "The number of nodes registered in the cluster.",
	}
	attrs["active_node_count"] = schema.Int64Attribute{
		Computed:    true,
		Description: "The number of nodes whose agent is currently connected.",
	}
	attrs["datacenters"] = schema.ListAttribute{
		Computed:    true,
		ElementType: types.StringType,
		Description: "Sorted, distinct data centres of the cluster nodes.",
	}
	attrs["versions"] = schema.ListAttribute{
		Computed:    true,
		ElementType: types.StringType,
		Description: "Sorted, distinct Cassandra/Kafka versions reported by the cluster nodes.",
	}

	resp.Schema = schema.Schema{
		Description: "Looks up a single cluster by name and returns its type, status, and node summary.",
		Attributes:  attrs,
	}
}

type clusterDataSourceData struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	Status          types.String `tfsdk:"status"`
	NodeCount       types.Int64  `tfsdk:"node_count"`
	ActiveNodeCount types.Int64  `tfsdk:"active_node_count"`
	Datacenters     types.List   `tfsdk:"datacenters"`
	Versions        types.List   `tfsdk:"versions"`
}

func (d *clusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clusterDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusters, err := d.client.ListClusters(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list clusters: %s", err))
		return
	}

	var matches []axonopsClient.ClusterSummary
	for _, c := range clusters {
		if c.Name == data.Name.ValueString() && matchesFilter(data.Type, c.Type) {
			matches = append(matches, c)
		}
	}

	switch len(matches) {
	case 0:
		msg := fmt.Sprintf("Cluster %q not found in organisation %q", data.Name.ValueString(), d.client.OrgId())
		if !data.Type.IsNull() {
			msg = fmt.Sprintf("%s cluster %q not found in organisation %q", data.Type.ValueString(), data.Name.ValueString(), d.client.OrgId())
		}
		resp.Diagnostics.AddError("Not Found", msg+", or the API key cannot access it.")
		return
	case 1:
	default:
		matchTypes := make([]string, 0, len(matches))
		for _, m := range matches {
			matchTypes = append(matchTypes, m.Type)
		}
		sort.Strings(matchTypes)
		resp.Diagnostics.AddError("Ambiguous Cluster Name",
			fmt.Sprintf("More than one cluster is named %q (types: %s). Set `type` to select one.",
				data.Name.ValueString(), strings.Join(matchTypes, ", ")))
		return
	}
	cluster := matches[0]

	nodes, err := d.client.GetClusterNodes(ctx, cluster.Type, cluster.Name)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read nodes of cluster %q: %s", cluster.Name, err))
		return
	}

	var active int64
	dcs := map[string]struct{}{}
	versions := map[string]struct{}{}
	for _, n := range nodes {
		if n.Active {
			active++
		}
		if n.DC != "" {
			dcs[n.DC] = struct{}{}
		}
		if v := n.Details["comp_releaseVersion"]; v != "" {
			versions[v] = struct{}{}
		}
	}

	data.ID = types.StringValue(clusterID(cluster.Type, cluster.Name))
	data.Type = types.StringValue(cluster.Type)
	data.Status = types.StringValue(clusterStatusName(cluster.Status))
	data.NodeCount = types.Int64Value(int64(len(nodes)))
	data.ActiveNodeCount = types.Int64Value(active)
	data.Datacenters = stringListValue(sortedKeys(dcs))
	data.Versions = stringListValue(sortedKeys(versions))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
