package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*cassandraScheduledRepairDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*cassandraScheduledRepairDataSource)(nil)

type cassandraScheduledRepairDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraScheduledRepairDataSource() datasource.DataSource {
	return &cassandraScheduledRepairDataSource{}
}

func (d *cassandraScheduledRepairDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *cassandraScheduledRepairDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_scheduled_repair"
}

func (d *cassandraScheduledRepairDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Cassandra scheduled repair configuration by tag.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Cassandra cluster.",
			},
			"tag": schema.StringAttribute{
				Required:    true,
				Description: "The tag identifying the scheduled repair.",
			},
			"repair_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the scheduled repair.",
			},
			"keyspace": schema.StringAttribute{
				Computed:    true,
				Description: "The keyspace to repair. Empty string means all keyspaces.",
			},
			"tables": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of tables to repair. Empty means all tables.",
			},
			"blacklisted_tables": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of tables to exclude from repair.",
			},
			"nodes": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of specific nodes to repair. Empty means all nodes.",
			},
			"segments_per_node": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of segments per node.",
			},
			"segmented": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether segmented repair is enabled.",
			},
			"incremental": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether incremental repair is enabled.",
			},
			"job_threads": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of job threads.",
			},
			"schedule_expr": schema.StringAttribute{
				Computed:    true,
				Description: "Cron expression for the repair schedule.",
			},
			"primary_range": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether primary range repair is enabled.",
			},
			"parallelism": schema.StringAttribute{
				Computed:    true,
				Description: "Repair parallelism mode.",
			},
			"optimise_streams": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether to optimise repair streams.",
			},
			"specific_data_centers": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of specific data centers to repair. Empty means all data centers.",
			},
			"skip_paxos": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether to skip Paxos repair.",
			},
			"paxos_only": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether to only run Paxos repair.",
			},
		},
	}
}

type cassandraScheduledRepairDataSourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Tag                 types.String `tfsdk:"tag"`
	RepairID            types.String `tfsdk:"repair_id"`
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
	OptimiseStreams     types.Bool   `tfsdk:"optimise_streams"`
	SpecificDataCenters types.List   `tfsdk:"specific_data_centers"`
	SkipPaxos           types.Bool   `tfsdk:"skip_paxos"`
	PaxosOnly           types.Bool   `tfsdk:"paxos_only"`
}

func (d *cassandraScheduledRepairDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data cassandraScheduledRepairDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	repairs, err := d.client.GetScheduledRepairs(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled repairs: %s", err))
		return
	}

	entry := axonopsClient.FindScheduledRepairByTag(repairs, data.Tag.ValueString())
	if entry == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("No scheduled repair found with tag '%s' in cluster '%s'", data.Tag.ValueString(), data.ClusterName.ValueString()))
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
