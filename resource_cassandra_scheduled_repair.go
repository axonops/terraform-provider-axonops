package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

var _ resource.Resource = (*cassandraScheduledRepairResource)(nil)
var _ resource.ResourceWithImportState = (*cassandraScheduledRepairResource)(nil)

type cassandraScheduledRepairResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraScheduledRepairResource() resource.Resource {
	return &cassandraScheduledRepairResource{}
}

func (r *cassandraScheduledRepairResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *cassandraScheduledRepairResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_scheduled_repair"
}

func (r *cassandraScheduledRepairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Cassandra scheduled repair configuration for a cluster. Updates are performed as delete-then-create since the API does not support in-place updates.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Cassandra cluster.",
			},
			"tag": schema.StringAttribute{
				Required:    true,
				Description: "A unique tag to identify this scheduled repair. Used for matching during updates and deletes.",
			},
			"keyspace": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The keyspace to repair. Empty string means all keyspaces.",
			},
			"tables": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "List of tables to repair. Empty means all tables.",
			},
			"blacklisted_tables": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "List of tables to exclude from repair.",
			},
			"nodes": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "List of specific nodes to repair. Empty means all nodes.",
			},
			"segments_per_node": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Description: "Number of segments per node. Default: 1",
			},
			"segmented": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to use segmented repair. Default: false",
			},
			"incremental": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to use incremental repair. Default: false",
			},
			"job_threads": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Description: "Number of job threads. Default: 1",
			},
			"schedule_expr": schema.StringAttribute{
				Required:    true,
				Description: "Cron expression for the repair schedule (e.g. '0 0 1 * *' for the first day of each month at midnight).",
			},
			"primary_range": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to use primary range repair. Default: false",
			},
			"parallelism": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Parallel"),
				Description: "Repair parallelism mode. Valid values: Parallel, Sequential, DC-Aware. Default: Parallel",
			},
			"optimise_streams": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to optimise repair streams. Default: false",
			},
			"specific_data_centers": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "List of specific data centers to repair. Empty means all data centers.",
			},
			"skip_paxos": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to skip Paxos repair. Mutually exclusive with paxos_only. Default: false",
			},
			"paxos_only": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to only run Paxos repair. Mutually exclusive with skip_paxos. Default: false",
			},
			"repair_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the scheduled repair assigned by AxonOps.",
			},
		},
	}
}

type cassandraScheduledRepairResourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Tag                 types.String `tfsdk:"tag"`
	Keyspace            types.String `tfsdk:"keyspace"`
	Tables              types.List   `tfsdk:"tables"`
	BlacklistedTables   types.List   `tfsdk:"blacklisted_tables"`
	Nodes               types.List   `tfsdk:"nodes"`
	SegmentsPerNode     types.Int64  `tfsdk:"segments_per_node"`
	Segmented           types.Bool   `tfsdk:"segmented"`
	Incremental         types.Bool   `tfsdk:"incremental"`
	JobThreads          types.Int64  `tfsdk:"job_threads"`
	ScheduleExpr        types.String `tfsdk:"schedule_expr"`
	PrimaryRange        types.Bool   `tfsdk:"primary_range"`
	Parallelism         types.String `tfsdk:"parallelism"`
	OptimiseStreams      types.Bool   `tfsdk:"optimise_streams"`
	SpecificDataCenters types.List   `tfsdk:"specific_data_centers"`
	SkipPaxos           types.Bool   `tfsdk:"skip_paxos"`
	PaxosOnly           types.Bool   `tfsdk:"paxos_only"`
	RepairID            types.String `tfsdk:"repair_id"`
}

func (r *cassandraScheduledRepairResource) buildParams(ctx context.Context, data *cassandraScheduledRepairResourceData, diagnostics *diag.Diagnostics) axonopsClient.ScheduledRepairParams {
	var tables, blacklisted, nodes, datacenters []string

	d := data.Tables.ElementsAs(ctx, &tables, false)
	diagnostics.Append(d...)
	d = data.BlacklistedTables.ElementsAs(ctx, &blacklisted, false)
	diagnostics.Append(d...)
	d = data.Nodes.ElementsAs(ctx, &nodes, false)
	diagnostics.Append(d...)
	d = data.SpecificDataCenters.ElementsAs(ctx, &datacenters, false)
	diagnostics.Append(d...)

	if tables == nil {
		tables = []string{}
	}
	if blacklisted == nil {
		blacklisted = []string{}
	}
	if nodes == nil {
		nodes = []string{}
	}
	if datacenters == nil {
		datacenters = []string{}
	}

	if data.SkipPaxos.ValueBool() && data.PaxosOnly.ValueBool() {
		diagnostics.AddError(
			"Invalid Configuration",
			"skip_paxos and paxos_only cannot both be true",
		)
		return axonopsClient.ScheduledRepairParams{}
	}

	paxos := "Default"
	if data.SkipPaxos.ValueBool() {
		paxos = "Skip Paxos"
	}
	if data.PaxosOnly.ValueBool() {
		paxos = "Paxos Only"
	}

	return axonopsClient.ScheduledRepairParams{
		Keyspace:            data.Keyspace.ValueString(),
		Tables:              tables,
		BlacklistedTables:   blacklisted,
		Nodes:               nodes,
		SegmentsPerNode:     int(data.SegmentsPerNode.ValueInt64()),
		Segmented:           data.Segmented.ValueBool(),
		Incremental:         data.Incremental.ValueBool(),
		JobThreads:          int(data.JobThreads.ValueInt64()),
		Schedule:            true,
		ScheduleExpr:        data.ScheduleExpr.ValueString(),
		PrimaryRange:        data.PrimaryRange.ValueBool(),
		Parallelism:         data.Parallelism.ValueString(),
		OptimiseStreams:      data.OptimiseStreams.ValueBool(),
		SpecificDataCenters: datacenters,
		Tag:                 data.Tag.ValueString(),
		Paxos:               paxos,
		SkipPaxos:           data.SkipPaxos.ValueBool(),
		PaxosOnly:           data.PaxosOnly.ValueBool(),
	}
}

func (r *cassandraScheduledRepairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cassandraScheduledRepairResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := r.buildParams(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.CreateScheduledRepair(data.ClusterName.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create scheduled repair: %s", err))
		return
	}

	// Fetch the created repair to get its ID
	repairs, err := r.client.GetScheduledRepairs(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled repairs after creation: %s", err))
		return
	}

	entry := axonopsClient.FindScheduledRepairByTag(repairs, data.Tag.ValueString())
	if entry == nil {
		resp.Diagnostics.AddError("Consistency Error",
			fmt.Sprintf("Scheduled repair was created but could not be found by tag %q", data.Tag.ValueString()))
		return
	}
	data.RepairID = types.StringValue(entry.ID)

	tflog.Info(ctx, "Created Cassandra scheduled repair resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraScheduledRepairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cassandraScheduledRepairResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	repairs, err := r.client.GetScheduledRepairs(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled repairs: %s", err))
		return
	}

	entry := axonopsClient.FindScheduledRepairByTag(repairs, data.Tag.ValueString())
	if entry == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.RepairID = types.StringValue(entry.ID)

	if len(entry.Params) > 0 {
		p := entry.Params[0]
		data.Keyspace = types.StringValue(p.Keyspace)
		data.SegmentsPerNode = types.Int64Value(int64(p.SegmentsPerNode))
		data.Segmented = types.BoolValue(p.Segmented)
		data.Incremental = types.BoolValue(p.Incremental)
		data.JobThreads = types.Int64Value(int64(p.JobThreads))
		data.ScheduleExpr = types.StringValue(p.ScheduleExpr)
		data.PrimaryRange = types.BoolValue(p.PrimaryRange)
		data.Parallelism = types.StringValue(p.Parallelism)
		data.OptimiseStreams = types.BoolValue(p.OptimiseStreams)
		data.SkipPaxos = types.BoolValue(p.SkipPaxos)
		data.PaxosOnly = types.BoolValue(p.PaxosOnly)
		data.Tag = types.StringValue(p.Tag)

		tables := p.Tables
		if tables == nil {
			tables = []string{}
		}
		data.Tables, diags = types.ListValueFrom(ctx, types.StringType, tables)
		resp.Diagnostics.Append(diags...)

		blacklisted := p.BlacklistedTables
		if blacklisted == nil {
			blacklisted = []string{}
		}
		data.BlacklistedTables, diags = types.ListValueFrom(ctx, types.StringType, blacklisted)
		resp.Diagnostics.Append(diags...)

		nodes := p.Nodes
		if nodes == nil {
			nodes = []string{}
		}
		data.Nodes, diags = types.ListValueFrom(ctx, types.StringType, nodes)
		resp.Diagnostics.Append(diags...)

		datacenters := p.SpecificDataCenters
		if datacenters == nil {
			datacenters = []string{}
		}
		data.SpecificDataCenters, diags = types.ListValueFrom(ctx, types.StringType, datacenters)
		resp.Diagnostics.Append(diags...)
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraScheduledRepairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data cassandraScheduledRepairResourceData
	var state cassandraScheduledRepairResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing repair by ID if we have one
	if state.RepairID.ValueString() != "" {
		err := r.client.DeleteScheduledRepair(state.ClusterName.ValueString(), state.RepairID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete existing scheduled repair during update: %s", err))
			return
		}
	}

	params := r.buildParams(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.CreateScheduledRepair(data.ClusterName.ValueString(), params)
	if err != nil {
		// The old repair was already deleted; clear state so Terraform knows
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Deleted existing repair but failed to create replacement: %s", err))
		return
	}

	// Fetch the new repair ID
	repairs, err := r.client.GetScheduledRepairs(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled repairs after update: %s", err))
		return
	}

	entry := axonopsClient.FindScheduledRepairByTag(repairs, data.Tag.ValueString())
	if entry == nil {
		resp.Diagnostics.AddError("Consistency Error",
			fmt.Sprintf("Scheduled repair was created but could not be found by tag %q", data.Tag.ValueString()))
		return
	}
	data.RepairID = types.StringValue(entry.ID)

	tflog.Info(ctx, "Updated Cassandra scheduled repair resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraScheduledRepairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cassandraScheduledRepairResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.RepairID.ValueString() != "" {
		err := r.client.DeleteScheduledRepair(data.ClusterName.ValueString(), data.RepairID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete scheduled repair: %s", err))
			return
		}
	} else {
		// Try to find by tag and delete
		repairs, err := r.client.GetScheduledRepairs(data.ClusterName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled repairs: %s", err))
			return
		}

		entry := axonopsClient.FindScheduledRepairByTag(repairs, data.Tag.ValueString())
		if entry != nil {
			err := r.client.DeleteScheduledRepair(data.ClusterName.ValueString(), entry.ID)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete scheduled repair: %s", err))
				return
			}
		}
	}

	tflog.Info(ctx, "Deleted Cassandra scheduled repair resource")
}

// ImportState imports an existing scheduled repair.
// Import ID format: cluster_name/tag
func (r *cassandraScheduledRepairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_name/tag, got: %s", req.ID),
		)
		return
	}

	clusterName := parts[0]
	tag := parts[1]

	repairs, err := r.client.GetScheduledRepairs(clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read scheduled repairs: %s", err))
		return
	}

	entry := axonopsClient.FindScheduledRepairByTag(repairs, tag)
	if entry == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("No scheduled repair found with tag '%s' in cluster '%s'", tag, clusterName))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag"), tag)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repair_id"), entry.ID)...)

	if len(entry.Params) > 0 {
		p := entry.Params[0]
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("keyspace"), p.Keyspace)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("segments_per_node"), int64(p.SegmentsPerNode))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("segmented"), p.Segmented)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("incremental"), p.Incremental)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("job_threads"), int64(p.JobThreads))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schedule_expr"), p.ScheduleExpr)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("primary_range"), p.PrimaryRange)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("parallelism"), p.Parallelism)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("optimise_streams"), p.OptimiseStreams)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("skip_paxos"), p.SkipPaxos)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("paxos_only"), p.PaxosOnly)...)

		tables := p.Tables
		if tables == nil {
			tables = []string{}
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tables"), tables)...)

		blacklisted := p.BlacklistedTables
		if blacklisted == nil {
			blacklisted = []string{}
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("blacklisted_tables"), blacklisted)...)

		nodes := p.Nodes
		if nodes == nil {
			nodes = []string{}
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("nodes"), nodes)...)

		datacenters := p.SpecificDataCenters
		if datacenters == nil {
			datacenters = []string{}
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("specific_data_centers"), datacenters)...)
	}

	tflog.Info(ctx, "Imported Cassandra scheduled repair", map[string]any{
		"cluster_name": clusterName,
		"tag":          tag,
	})
}
