package main

import (
	"context"
	"fmt"
	"sort"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*clustersDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*clustersDataSource)(nil)

type clustersDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewClustersDataSource() datasource.DataSource {
	return &clustersDataSource{}
}

func (d *clustersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *clustersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clusters"
}

// clusterStatusName maps the AxonOps alert RAG level to a readable string.
func clusterStatusName(status int) string {
	switch status {
	case 0:
		return "green"
	case 1:
		return "amber"
	case 2:
		return "red"
	default:
		return "unknown"
	}
}

// clusterID is the stable identifier used for clusters: "<type>/<name>".
func clusterID(clusterType, clusterName string) string {
	return clusterType + "/" + clusterName
}

func (d *clustersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the clusters in the organisation that the API key can access.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Only return clusters of this type (cassandra, kafka, or dse).",
				Validators:  []validator.String{clusterTypeValidator()},
			},
			"names": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Names of the matching clusters, sorted by type then name.",
			},
			"clusters": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Matching clusters, sorted by type then name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: clusterSummaryAttributes(),
				},
			},
		},
	}
}

// clusterSummaryAttributes is shared by axonops_clusters entries and
// axonops_cluster.
func clusterSummaryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "The cluster identifier, `<type>/<name>`.",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "The cluster name.",
		},
		"type": schema.StringAttribute{
			Computed:    true,
			Description: "The cluster type (cassandra, kafka, or dse).",
		},
		"status": schema.StringAttribute{
			Computed:    true,
			Description: "The cluster alert status: `green`, `amber`, or `red`.",
		},
	}
}

type clustersDataSourceData struct {
	Type     types.String       `tfsdk:"type"`
	Names    types.List         `tfsdk:"names"`
	Clusters []clusterListEntry `tfsdk:"clusters"`
}

type clusterListEntry struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	Status types.String `tfsdk:"status"`
}

func (d *clustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clustersDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusters, err := d.client.ListClusters(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list clusters: %s", err))
		return
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Type != clusters[j].Type {
			return clusters[i].Type < clusters[j].Type
		}
		return clusters[i].Name < clusters[j].Name
	})

	names := []string{}
	entries := []clusterListEntry{}
	for _, c := range clusters {
		if !matchesFilter(data.Type, c.Type) {
			continue
		}
		names = append(names, c.Name)
		entries = append(entries, clusterListEntry{
			ID:     types.StringValue(clusterID(c.Type, c.Name)),
			Name:   types.StringValue(c.Name),
			Type:   types.StringValue(c.Type),
			Status: types.StringValue(clusterStatusName(c.Status)),
		})
	}
	data.Names = stringListValue(names)
	data.Clusters = entries

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
