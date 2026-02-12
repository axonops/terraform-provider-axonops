package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "axonops-tf/client"

	"github.com/google/uuid"
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

var _ resource.Resource = (*cassandraBackupResource)(nil)
var _ resource.ResourceWithImportState = (*cassandraBackupResource)(nil)

type cassandraBackupResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraBackupResource() resource.Resource {
	return &cassandraBackupResource{}
}

func (r *cassandraBackupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *cassandraBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_backup"
}

func (r *cassandraBackupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cassandra backup schedule.",
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
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the backup (auto-generated).",
			},
			"tag": schema.StringAttribute{
				Required:    true,
				Description: "Unique name/tag for the backup.",
			},
			"datacenters": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "List of datacenters to back up.",
			},
			"schedule": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether scheduling is enabled. Default: true",
			},
			"schedule_expr": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0 1 * * *"),
				Description: "Cron expression for backup schedule. Default: 0 1 * * *",
			},
			"local_retention": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("10d"),
				Description: "Local backup retention duration. Default: 10d",
			},
			"remote": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to enable remote backup. Default: false",
			},
			"remote_type": schema.StringAttribute{
				Optional:    true,
				Description: "Remote storage type: s3, sftp, azure.",
			},
			"remote_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path on the remote storage.",
			},
			"remote_retention": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("60d"),
				Description: "Remote backup retention duration. Default: 60d",
			},
			"remote_config": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Remote storage configuration as key=value pairs separated by newlines.",
			},
			"timeout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("10h"),
				Description: "Backup operation timeout. Default: 10h",
			},
			"transfers": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Description: "Number of parallel transfers. Default: 1",
			},
			"tps_limit": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(50),
				Description: "Throughput per second limit. Default: 50",
			},
			"bw_limit": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Bandwidth limit.",
			},
			"keyspaces": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "Keyspaces to backup. Empty means all keyspaces.",
			},
			"tables": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "Tables to backup (format: keyspace.table). Empty means all tables.",
			},
			"nodes": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "Specific node IDs to backup. Empty means all nodes.",
			},
		},
	}
}

type cassandraBackupResourceData struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	ClusterType     types.String `tfsdk:"cluster_type"`
	ID              types.String `tfsdk:"id"`
	Tag             types.String `tfsdk:"tag"`
	Datacenters     types.List   `tfsdk:"datacenters"`
	Schedule        types.Bool   `tfsdk:"schedule"`
	ScheduleExpr    types.String `tfsdk:"schedule_expr"`
	LocalRetention  types.String `tfsdk:"local_retention"`
	Remote          types.Bool   `tfsdk:"remote"`
	RemoteType      types.String `tfsdk:"remote_type"`
	RemotePath      types.String `tfsdk:"remote_path"`
	RemoteRetention types.String `tfsdk:"remote_retention"`
	RemoteConfig    types.String `tfsdk:"remote_config"`
	Timeout         types.String `tfsdk:"timeout"`
	Transfers       types.Int64  `tfsdk:"transfers"`
	TpsLimit        types.Int64  `tfsdk:"tps_limit"`
	BwLimit         types.String `tfsdk:"bw_limit"`
	Keyspaces       types.List   `tfsdk:"keyspaces"`
	Tables          types.List   `tfsdk:"tables"`
	Nodes           types.List   `tfsdk:"nodes"`
}

func (r *cassandraBackupResource) buildBackup(ctx context.Context, data *cassandraBackupResourceData, resp *resource.CreateResponse) *axonopsClient.CassandraBackup {
	var datacenters, keyspaces, tables, nodes []string

	diags := data.Datacenters.ElementsAs(ctx, &datacenters, false)
	resp.Diagnostics.Append(diags...)
	diags = data.Keyspaces.ElementsAs(ctx, &keyspaces, false)
	resp.Diagnostics.Append(diags...)
	diags = data.Tables.ElementsAs(ctx, &tables, false)
	resp.Diagnostics.Append(diags...)
	diags = data.Nodes.ElementsAs(ctx, &nodes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	if keyspaces == nil {
		keyspaces = []string{}
	}
	if tables == nil {
		tables = []string{}
	}
	if nodes == nil {
		nodes = []string{}
	}

	backup := &axonopsClient.CassandraBackup{
		ID:                     data.ID.ValueString(),
		Tag:                    data.Tag.ValueString(),
		LocalRetentionDuration: data.LocalRetention.ValueString(),
		Remote:                 data.Remote.ValueBool(),
		Timeout:                data.Timeout.ValueString(),
		Transfers:              int(data.Transfers.ValueInt64()),
		TpsLimit:               int(data.TpsLimit.ValueInt64()),
		BwLimit:                data.BwLimit.ValueString(),
		Datacenters:            datacenters,
		Nodes:                  nodes,
		Tables:                 tables,
		Keyspaces:              keyspaces,
		AllTables:              len(tables) == 0,
		AllNodes:               len(nodes) == 0,
		Schedule:               data.Schedule.ValueBool(),
		ScheduleExpr:           data.ScheduleExpr.ValueString(),
	}

	if data.Remote.ValueBool() {
		backup.RemoteType = data.RemoteType.ValueString()
		backup.RemotePath = data.RemotePath.ValueString()
		backup.RemoteRetentionDuration = data.RemoteRetention.ValueString()
		backup.RemoteConfig = data.RemoteConfig.ValueString()
	}

	return backup
}

func (r *cassandraBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cassandraBackupResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newID := uuid.New().String()
	data.ID = types.StringValue(newID)

	// Reuse the common build logic (using a CreateResponse wrapper)
	var datacenters, keyspaces, tables, nodes []string

	diags = data.Datacenters.ElementsAs(ctx, &datacenters, false)
	resp.Diagnostics.Append(diags...)
	diags = data.Keyspaces.ElementsAs(ctx, &keyspaces, false)
	resp.Diagnostics.Append(diags...)
	diags = data.Tables.ElementsAs(ctx, &tables, false)
	resp.Diagnostics.Append(diags...)
	diags = data.Nodes.ElementsAs(ctx, &nodes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if keyspaces == nil {
		keyspaces = []string{}
	}
	if tables == nil {
		tables = []string{}
	}
	if nodes == nil {
		nodes = []string{}
	}

	backup := axonopsClient.CassandraBackup{
		ID:                     newID,
		Tag:                    data.Tag.ValueString(),
		LocalRetentionDuration: data.LocalRetention.ValueString(),
		Remote:                 data.Remote.ValueBool(),
		Timeout:                data.Timeout.ValueString(),
		Transfers:              int(data.Transfers.ValueInt64()),
		TpsLimit:               int(data.TpsLimit.ValueInt64()),
		BwLimit:                data.BwLimit.ValueString(),
		Datacenters:            datacenters,
		Nodes:                  nodes,
		Tables:                 tables,
		Keyspaces:              keyspaces,
		AllTables:              len(tables) == 0,
		AllNodes:               len(nodes) == 0,
		Schedule:               data.Schedule.ValueBool(),
		ScheduleExpr:           data.ScheduleExpr.ValueString(),
	}

	if data.Remote.ValueBool() {
		backup.RemoteType = data.RemoteType.ValueString()
		backup.RemotePath = data.RemotePath.ValueString()
		backup.RemoteRetentionDuration = data.RemoteRetention.ValueString()
		backup.RemoteConfig = data.RemoteConfig.ValueString()
	}

	err := r.client.CreateCassandraBackup(data.ClusterType.ValueString(), data.ClusterName.ValueString(), backup)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup: %s", err))
		return
	}

	tflog.Info(ctx, "Created Cassandra backup resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cassandraBackupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	backups, err := r.client.GetCassandraBackups(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backups: %s", err))
		return
	}

	// Find backup by tag
	var found *axonopsClient.CassandraBackup
	for _, b := range backups {
		if b.Tag == data.Tag.ValueString() {
			found = &b
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(found.ID)
	data.LocalRetention = types.StringValue(found.LocalRetentionDuration)
	data.Remote = types.BoolValue(found.Remote)
	data.Schedule = types.BoolValue(found.Schedule)
	data.ScheduleExpr = types.StringValue(found.ScheduleExpr)
	data.Timeout = types.StringValue(found.Timeout)
	data.Transfers = types.Int64Value(int64(found.Transfers))
	data.TpsLimit = types.Int64Value(int64(found.TpsLimit))
	data.BwLimit = types.StringValue(found.BwLimit)

	if found.Remote {
		data.RemoteType = types.StringValue(found.RemoteType)
		data.RemotePath = types.StringValue(found.RemotePath)
		data.RemoteRetention = types.StringValue(found.RemoteRetentionDuration)
		data.RemoteConfig = types.StringValue(found.RemoteConfig)
	}

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

func (r *cassandraBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData cassandraBackupResourceData
	var stateData cassandraBackupResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the old backup
	err := r.client.DeleteCassandraBackup(stateData.ClusterType.ValueString(), stateData.ClusterName.ValueString(), []string{stateData.ID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete old backup for update: %s", err))
		return
	}

	// Create new backup with new ID
	newID := uuid.New().String()
	planData.ID = types.StringValue(newID)

	var datacenters, keyspaces, tables, nodes []string

	diags = planData.Datacenters.ElementsAs(ctx, &datacenters, false)
	resp.Diagnostics.Append(diags...)
	diags = planData.Keyspaces.ElementsAs(ctx, &keyspaces, false)
	resp.Diagnostics.Append(diags...)
	diags = planData.Tables.ElementsAs(ctx, &tables, false)
	resp.Diagnostics.Append(diags...)
	diags = planData.Nodes.ElementsAs(ctx, &nodes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if keyspaces == nil {
		keyspaces = []string{}
	}
	if tables == nil {
		tables = []string{}
	}
	if nodes == nil {
		nodes = []string{}
	}

	backup := axonopsClient.CassandraBackup{
		ID:                     newID,
		Tag:                    planData.Tag.ValueString(),
		LocalRetentionDuration: planData.LocalRetention.ValueString(),
		Remote:                 planData.Remote.ValueBool(),
		Timeout:                planData.Timeout.ValueString(),
		Transfers:              int(planData.Transfers.ValueInt64()),
		TpsLimit:               int(planData.TpsLimit.ValueInt64()),
		BwLimit:                planData.BwLimit.ValueString(),
		Datacenters:            datacenters,
		Nodes:                  nodes,
		Tables:                 tables,
		Keyspaces:              keyspaces,
		AllTables:              len(tables) == 0,
		AllNodes:               len(nodes) == 0,
		Schedule:               planData.Schedule.ValueBool(),
		ScheduleExpr:           planData.ScheduleExpr.ValueString(),
	}

	if planData.Remote.ValueBool() {
		backup.RemoteType = planData.RemoteType.ValueString()
		backup.RemotePath = planData.RemotePath.ValueString()
		backup.RemoteRetentionDuration = planData.RemoteRetention.ValueString()
		backup.RemoteConfig = planData.RemoteConfig.ValueString()
	}

	err = r.client.CreateCassandraBackup(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), backup)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create updated backup: %s", err))
		return
	}

	tflog.Info(ctx, "Updated Cassandra backup resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cassandraBackupResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCassandraBackup(data.ClusterType.ValueString(), data.ClusterName.ValueString(), []string{data.ID.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete backup: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted Cassandra backup resource")
}

// ImportState imports an existing backup.
// Import ID format: cluster_type/cluster_name/tag
func (r *cassandraBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/tag, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]
	tag := parts[2]

	backups, err := r.client.GetCassandraBackups(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read backups: %s", err))
		return
	}

	var found *axonopsClient.CassandraBackup
	for _, b := range backups {
		if b.Tag == tag {
			found = &b
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Backup with tag %s not found in cluster %s/%s", tag, clusterType, clusterName))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag"), found.Tag)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schedule"), found.Schedule)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schedule_expr"), found.ScheduleExpr)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("local_retention"), found.LocalRetentionDuration)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), found.Remote)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote_type"), found.RemoteType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote_path"), found.RemotePath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote_retention"), found.RemoteRetentionDuration)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote_config"), found.RemoteConfig)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timeout"), found.Timeout)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("transfers"), int64(found.Transfers))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tps_limit"), int64(found.TpsLimit))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bw_limit"), found.BwLimit)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("datacenters"), found.Datacenters)...)

	keyspaces := found.Keyspaces
	if keyspaces == nil {
		keyspaces = []string{}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("keyspaces"), keyspaces)...)

	tables := found.Tables
	if tables == nil {
		tables = []string{}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tables"), tables)...)

	nodes := found.Nodes
	if nodes == nil {
		nodes = []string{}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("nodes"), nodes)...)

	tflog.Info(ctx, fmt.Sprintf("Imported Cassandra backup %s from cluster %s/%s", tag, clusterType, clusterName))
}
