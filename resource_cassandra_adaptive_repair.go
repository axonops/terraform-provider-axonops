package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*cassandraAdaptiveRepairResource)(nil)
var _ resource.ResourceWithImportState = (*cassandraAdaptiveRepairResource)(nil)

type cassandraAdaptiveRepairResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraAdaptiveRepairResource() resource.Resource {
	return &cassandraAdaptiveRepairResource{}
}

func (r *cassandraAdaptiveRepairResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *cassandraAdaptiveRepairResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_adaptive_repair"
}

func (r *cassandraAdaptiveRepairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Cassandra adaptive repair settings for a cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("cassandra"),
				Description: "The cluster type (cassandra or dse). Default: cassandra",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether adaptive repair is enabled. Default: true",
			},
			"parallelism": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10),
				Description: "Number of tables to repair concurrently. Default: 10",
			},
			"gc_grace_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(86400),
				Description: "GC grace period threshold in seconds. Default: 86400",
			},
			"blacklisted_tables": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "List of tables to exclude from repair.",
			},
			"filter_twcs_tables": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to exclude TWCS (TimeWindowCompactionStrategy) tables. Default: true",
			},
			"segment_retries": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
				Description: "Maximum retry attempts per segment. Default: 3",
			},
			"segments_per_vnode": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Description: "Number of segments per vnode. Default: 1",
			},
			"segment_target_size_mb": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(256),
				Description: "Target segment size in MB. Default: 256",
			},
		},
	}
}

type cassandraAdaptiveRepairResourceData struct {
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

func (r *cassandraAdaptiveRepairResource) buildSettings(ctx context.Context, data *cassandraAdaptiveRepairResourceData, diags *[]interface{}) axonopsClient.AdaptiveRepairSettings {
	var blacklisted []string
	data.BlacklistedTables.ElementsAs(ctx, &blacklisted, false)
	if blacklisted == nil {
		blacklisted = []string{}
	}

	return axonopsClient.AdaptiveRepairSettings{
		Active:              data.Active.ValueBool(),
		GcGraceThreshold:    int(data.GcGraceThreshold.ValueInt64()),
		TableParallelism:    int(data.Parallelism.ValueInt64()),
		BlacklistedTables:   blacklisted,
		FilterTWCSTables:    data.FilterTwcsTables.ValueBool(),
		SegmentRetries:      int(data.SegmentRetries.ValueInt64()),
		SegmentsPerVnode:    int(data.SegmentsPerVnode.ValueInt64()),
		SegmentTargetSizeMB: int(data.SegmentTargetSizeMB.ValueInt64()),
	}
}

func (r *cassandraAdaptiveRepairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cassandraAdaptiveRepairResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var blacklisted []string
	diags = data.BlacklistedTables.ElementsAs(ctx, &blacklisted, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if blacklisted == nil {
		blacklisted = []string{}
	}

	settings := axonopsClient.AdaptiveRepairSettings{
		Active:              data.Active.ValueBool(),
		GcGraceThreshold:    int(data.GcGraceThreshold.ValueInt64()),
		TableParallelism:    int(data.Parallelism.ValueInt64()),
		BlacklistedTables:   blacklisted,
		FilterTWCSTables:    data.FilterTwcsTables.ValueBool(),
		SegmentRetries:      int(data.SegmentRetries.ValueInt64()),
		SegmentsPerVnode:    int(data.SegmentsPerVnode.ValueInt64()),
		SegmentTargetSizeMB: int(data.SegmentTargetSizeMB.ValueInt64()),
	}

	err := r.client.UpdateCassandraAdaptiveRepair(data.ClusterType.ValueString(), data.ClusterName.ValueString(), settings)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set adaptive repair settings: %s", err))
		return
	}

	tflog.Info(ctx, "Created Cassandra adaptive repair resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraAdaptiveRepairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cassandraAdaptiveRepairResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.GetCassandraAdaptiveRepair(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read adaptive repair settings: %s", err))
		return
	}

	data.Active = types.BoolValue(settings.Active)
	data.Parallelism = types.Int64Value(int64(settings.TableParallelism))
	data.GcGraceThreshold = types.Int64Value(int64(settings.GcGraceThreshold))
	data.FilterTwcsTables = types.BoolValue(settings.FilterTWCSTables)
	data.SegmentRetries = types.Int64Value(int64(settings.SegmentRetries))
	data.SegmentsPerVnode = types.Int64Value(int64(settings.SegmentsPerVnode))
	data.SegmentTargetSizeMB = types.Int64Value(int64(settings.SegmentTargetSizeMB))

	if settings.BlacklistedTables == nil {
		settings.BlacklistedTables = []string{}
	}
	data.BlacklistedTables, diags = types.ListValueFrom(ctx, types.StringType, settings.BlacklistedTables)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraAdaptiveRepairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data cassandraAdaptiveRepairResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var blacklisted []string
	diags = data.BlacklistedTables.ElementsAs(ctx, &blacklisted, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if blacklisted == nil {
		blacklisted = []string{}
	}

	settings := axonopsClient.AdaptiveRepairSettings{
		Active:              data.Active.ValueBool(),
		GcGraceThreshold:    int(data.GcGraceThreshold.ValueInt64()),
		TableParallelism:    int(data.Parallelism.ValueInt64()),
		BlacklistedTables:   blacklisted,
		FilterTWCSTables:    data.FilterTwcsTables.ValueBool(),
		SegmentRetries:      int(data.SegmentRetries.ValueInt64()),
		SegmentsPerVnode:    int(data.SegmentsPerVnode.ValueInt64()),
		SegmentTargetSizeMB: int(data.SegmentTargetSizeMB.ValueInt64()),
	}

	err := r.client.UpdateCassandraAdaptiveRepair(data.ClusterType.ValueString(), data.ClusterName.ValueString(), settings)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update adaptive repair settings: %s", err))
		return
	}

	tflog.Info(ctx, "Updated Cassandra adaptive repair resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraAdaptiveRepairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cassandraAdaptiveRepairResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset to defaults with Active=false
	settings := axonopsClient.AdaptiveRepairSettings{
		Active:              false,
		GcGraceThreshold:    86400,
		TableParallelism:    10,
		BlacklistedTables:   []string{},
		FilterTWCSTables:    true,
		SegmentRetries:      3,
		SegmentsPerVnode:    1,
		SegmentTargetSizeMB: 256,
	}

	err := r.client.UpdateCassandraAdaptiveRepair(data.ClusterType.ValueString(), data.ClusterName.ValueString(), settings)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset adaptive repair settings: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted (reset) Cassandra adaptive repair resource")
}

// ImportState imports existing adaptive repair settings.
// Import ID format: cluster_type/cluster_name
func (r *cassandraAdaptiveRepairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]

	settings, err := r.client.GetCassandraAdaptiveRepair(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read adaptive repair settings: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("active"), settings.Active)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("parallelism"), int64(settings.TableParallelism))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("gc_grace_threshold"), int64(settings.GcGraceThreshold))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("filter_twcs_tables"), settings.FilterTWCSTables)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("segment_retries"), int64(settings.SegmentRetries))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("segments_per_vnode"), int64(settings.SegmentsPerVnode))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("segment_target_size_mb"), int64(settings.SegmentTargetSizeMB))...)

	blacklisted := settings.BlacklistedTables
	if blacklisted == nil {
		blacklisted = []string{}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("blacklisted_tables"), blacklisted)...)

	tflog.Info(ctx, fmt.Sprintf("Imported Cassandra adaptive repair settings for cluster %s/%s", clusterType, clusterName))
}
