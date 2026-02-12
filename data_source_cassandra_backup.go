package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*cassandraBackupDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*cassandraBackupDataSource)(nil)

type cassandraBackupDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraBackupDataSource() datasource.DataSource {
	return &cassandraBackupDataSource{}
}

func (d *cassandraBackupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *cassandraBackupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_backup"
}

func (d *cassandraBackupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Cassandra backup schedule.",
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
			"tag": schema.StringAttribute{
				Required:    true,
				Description: "Unique name/tag for the backup.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the backup.",
			},
			"datacenters": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of datacenters.",
			},
			"schedule": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether scheduling is enabled.",
			},
			"schedule_expr": schema.StringAttribute{
				Computed:    true,
				Description: "Cron expression for backup schedule.",
			},
			"local_retention": schema.StringAttribute{
				Computed:    true,
				Description: "Local backup retention duration.",
			},
			"remote": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether remote backup is enabled.",
			},
			"remote_type": schema.StringAttribute{
				Computed:    true,
				Description: "Remote storage type.",
			},
			"remote_path": schema.StringAttribute{
				Computed:    true,
				Description: "Path on the remote storage.",
			},
			"remote_retention": schema.StringAttribute{
				Computed:    true,
				Description: "Remote backup retention duration.",
			},
			"timeout": schema.StringAttribute{
				Computed:    true,
				Description: "Backup operation timeout.",
			},
			"transfers": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of parallel transfers.",
			},
			"tps_limit": schema.Int64Attribute{
				Computed:    true,
				Description: "Throughput per second limit.",
			},
			"bw_limit": schema.StringAttribute{
				Computed:    true,
				Description: "Bandwidth limit.",
			},
			"keyspaces": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Keyspaces to backup.",
			},
			"tables": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Tables to backup.",
			},
			"nodes": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Specific node IDs.",
			},
		},
	}
}

type cassandraBackupDataSourceData struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	ClusterType     types.String `tfsdk:"cluster_type"`
	Tag             types.String `tfsdk:"tag"`
	ID              types.String `tfsdk:"id"`
	Datacenters     types.List   `tfsdk:"datacenters"`
	Schedule        types.Bool   `tfsdk:"schedule"`
	ScheduleExpr    types.String `tfsdk:"schedule_expr"`
	LocalRetention  types.String `tfsdk:"local_retention"`
	Remote          types.Bool   `tfsdk:"remote"`
	RemoteType      types.String `tfsdk:"remote_type"`
	RemotePath      types.String `tfsdk:"remote_path"`
	RemoteRetention types.String `tfsdk:"remote_retention"`
	Timeout         types.String `tfsdk:"timeout"`
	Transfers       types.Int64  `tfsdk:"transfers"`
	TpsLimit        types.Int64  `tfsdk:"tps_limit"`
	BwLimit         types.String `tfsdk:"bw_limit"`
	Keyspaces       types.List   `tfsdk:"keyspaces"`
	Tables          types.List   `tfsdk:"tables"`
	Nodes           types.List   `tfsdk:"nodes"`
}

func (d *cassandraBackupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data cassandraBackupDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterType := data.ClusterType.ValueString()
	if clusterType == "" {
		clusterType = "cassandra"
	}

	backups, err := d.client.GetCassandraBackups(clusterType, data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backups: %s", err))
		return
	}

	var found *axonopsClient.CassandraBackup
	for _, b := range backups {
		if b.Tag == data.Tag.ValueString() {
			found = &b
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Backup with tag %s not found", data.Tag.ValueString()))
		return
	}

	data.ClusterType = types.StringValue(clusterType)
	data.ID = types.StringValue(found.ID)
	data.LocalRetention = types.StringValue(found.LocalRetentionDuration)
	data.Remote = types.BoolValue(found.Remote)
	data.Schedule = types.BoolValue(found.Schedule)
	data.ScheduleExpr = types.StringValue(found.ScheduleExpr)
	data.Timeout = types.StringValue(found.Timeout)
	data.Transfers = types.Int64Value(int64(found.Transfers))
	data.TpsLimit = types.Int64Value(int64(found.TpsLimit))
	data.BwLimit = types.StringValue(found.BwLimit)
	data.RemoteType = types.StringValue(found.RemoteType)
	data.RemotePath = types.StringValue(found.RemotePath)
	data.RemoteRetention = types.StringValue(found.RemoteRetentionDuration)

	data.Datacenters, diags = types.ListValueFrom(ctx, types.StringType, found.Datacenters)
	resp.Diagnostics.Append(diags...)

	keyspaces := found.Keyspaces
	if keyspaces == nil {
		keyspaces = []string{}
	}
	data.Keyspaces, diags = types.ListValueFrom(ctx, types.StringType, keyspaces)
	resp.Diagnostics.Append(diags...)

	tables := found.Tables
	if tables == nil {
		tables = []string{}
	}
	data.Tables, diags = types.ListValueFrom(ctx, types.StringType, tables)
	resp.Diagnostics.Append(diags...)

	nodes := found.Nodes
	if nodes == nil {
		nodes = []string{}
	}
	data.Nodes, diags = types.ListValueFrom(ctx, types.StringType, nodes)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
