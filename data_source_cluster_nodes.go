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

var _ datasource.DataSource = (*clusterNodesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*clusterNodesDataSource)(nil)

type clusterNodesDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewClusterNodesDataSource() datasource.DataSource {
	return &clusterNodesDataSource{}
}

func (d *clusterNodesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *clusterNodesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_nodes"
}

func (d *clusterNodesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the nodes (agents) registered in a cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
				Validators:  []validator.String{clusterTypeValidator()},
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Description: "Only return nodes with this agent status. Valid values: `up`, `down`.",
				Validators:  []validator.String{stringvalidator.OneOf("up", "down")},
			},
			"datacenter": schema.StringAttribute{
				Optional:    true,
				Description: "Only return nodes in this data centre.",
			},
			"host_ids": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Host IDs of the matching nodes, in the same order as `nodes`.",
			},
			"nodes": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Matching nodes, sorted by data centre, rack, then host IP.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host_id": schema.StringAttribute{
							Computed:    true,
							Description: "The AxonOps host ID of the node.",
						},
						"host_ip": schema.StringAttribute{
							Computed:    true,
							Description: "The node IP address.",
						},
						"hostname": schema.StringAttribute{
							Computed:    true,
							Description: "The human-readable node identifier (usually the hostname).",
						},
						"datacenter": schema.StringAttribute{
							Computed:    true,
							Description: "The data centre of the node.",
						},
						"rack": schema.StringAttribute{
							Computed:    true,
							Description: "The rack of the node.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The agent status: `up` when connected, otherwise `down`.",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "The Cassandra/Kafka version running on the node.",
						},
						"agent_version": schema.StringAttribute{
							Computed:    true,
							Description: "The AxonOps agent version.",
						},
						"node_type": schema.StringAttribute{
							Computed:    true,
							Description: "The node role reported by the agent (for example a Kafka broker or KRaft controller). Empty when not reported.",
						},
					},
				},
			},
		},
	}
}

type clusterNodesDataSourceData struct {
	ClusterName types.String            `tfsdk:"cluster_name"`
	ClusterType types.String            `tfsdk:"cluster_type"`
	Status      types.String            `tfsdk:"status"`
	Datacenter  types.String            `tfsdk:"datacenter"`
	HostIDs     types.List              `tfsdk:"host_ids"`
	Nodes       []clusterNodesListEntry `tfsdk:"nodes"`
}

type clusterNodesListEntry struct {
	HostID       types.String `tfsdk:"host_id"`
	HostIP       types.String `tfsdk:"host_ip"`
	Hostname     types.String `tfsdk:"hostname"`
	Datacenter   types.String `tfsdk:"datacenter"`
	Rack         types.String `tfsdk:"rack"`
	Status       types.String `tfsdk:"status"`
	Version      types.String `tfsdk:"version"`
	AgentVersion types.String `tfsdk:"agent_version"`
	NodeType     types.String `tfsdk:"node_type"`
}

func nodeStatus(n axonopsClient.ClusterNodeInfo) string {
	if n.Active {
		return "up"
	}
	return "down"
}

func (d *clusterNodesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clusterNodesDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodes, err := d.client.GetClusterNodes(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list cluster nodes: %s", err))
		return
	}

	sort.Slice(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.DC != b.DC {
			return a.DC < b.DC
		}
		if a.Details["rack"] != b.Details["rack"] {
			return a.Details["rack"] < b.Details["rack"]
		}
		if a.HostIP != b.HostIP {
			return a.HostIP < b.HostIP
		}
		return a.HostID < b.HostID
	})

	hostIDs := []string{}
	entries := []clusterNodesListEntry{}
	for _, n := range nodes {
		status := nodeStatus(n)
		if !matchesFilter(data.Status, status) || !matchesFilter(data.Datacenter, n.DC) {
			continue
		}
		hostIDs = append(hostIDs, n.HostID)
		entries = append(entries, clusterNodesListEntry{
			HostID:       types.StringValue(n.HostID),
			HostIP:       types.StringValue(n.HostIP),
			Hostname:     types.StringValue(n.Details["human_readable_identifier"]),
			Datacenter:   types.StringValue(n.DC),
			Rack:         types.StringValue(n.Details["rack"]),
			Status:       types.StringValue(status),
			Version:      types.StringValue(n.Details["comp_releaseVersion"]),
			AgentVersion: types.StringValue(n.Details["agent_version"]),
			NodeType:     types.StringValue(n.Details["node_type"]),
		})
	}
	data.HostIDs = stringListValue(hostIDs)
	data.Nodes = entries

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
