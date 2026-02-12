package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*cassandraAdaptiveRepairDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*cassandraAdaptiveRepairDataSource)(nil)

type cassandraAdaptiveRepairDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraAdaptiveRepairDataSource() datasource.DataSource {
	return &cassandraAdaptiveRepairDataSource{}
}

func (d *cassandraAdaptiveRepairDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *cassandraAdaptiveRepairDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_adaptive_repair"
}

func (d *cassandraAdaptiveRepairDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Cassandra adaptive repair settings for a cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The cluster type (cassandra or dse). Default: cassandra",
			},
			"active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether adaptive repair is enabled.",
			},
			"parallelism": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of tables to repair concurrently.",
			},
			"gc_grace_threshold": schema.Int64Attribute{
				Computed:    true,
				Description: "GC grace period threshold in seconds.",
			},
			"blacklisted_tables": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of tables excluded from repair.",
			},
			"filter_twcs_tables": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether TWCS tables are excluded.",
			},
			"segment_retries": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum retry attempts per segment.",
			},
			"segments_per_vnode": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of segments per vnode.",
			},
			"segment_target_size_mb": schema.Int64Attribute{
				Computed:    true,
				Description: "Target segment size in MB.",
			},
		},
	}
}

type cassandraAdaptiveRepairDataSourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	ClusterType         types.String `tfsdk:"cluster_type"`
	Active              types.Bool   `tfsdk:"active"`
	Parallelism         types.Int64  `tfsdk:"parallelism"`
	GcGraceThreshold    types.Int64  `tfsdk:"gc_grace_threshold"`
	BlacklistedTables   types.List   `tfsdk:"blacklisted_tables"`
	FilterTwcsTables    types.Bool   `tfsdk:"filter_twcs_tables"`
	SegmentRetries      types.Int64  `tfsdk:"segment_retries"`
	SegmentsPerVnode    types.Int64  `tfsdk:"segments_per_vnode"`
	SegmentTargetSizeMB types.Int64  `tfsdk:"segment_target_size_mb"`
}

func (d *cassandraAdaptiveRepairDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data cassandraAdaptiveRepairDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterType := data.ClusterType.ValueString()
	if clusterType == "" {
		clusterType = "cassandra"
	}

	settings, err := d.client.GetCassandraAdaptiveRepair(clusterType, data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read adaptive repair settings: %s", err))
		return
	}

	data.ClusterType = types.StringValue(clusterType)
	data.Active = types.BoolValue(settings.Active)
	data.Parallelism = types.Int64Value(int64(settings.TableParallelism))
	data.GcGraceThreshold = types.Int64Value(int64(settings.GcGraceThreshold))
	data.FilterTwcsTables = types.BoolValue(settings.FilterTWCSTables)
	data.SegmentRetries = types.Int64Value(int64(settings.SegmentRetries))
	data.SegmentsPerVnode = types.Int64Value(int64(settings.SegmentsPerVnode))
	data.SegmentTargetSizeMB = types.Int64Value(int64(settings.SegmentTargetSizeMB))

	blacklisted := settings.BlacklistedTables
	if blacklisted == nil {
		blacklisted = []string{}
	}
	data.BlacklistedTables, diags = types.ListValueFrom(ctx, types.StringType, blacklisted)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
