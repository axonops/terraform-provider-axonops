package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// commitLogRemoteTypes lists the storage backends the AxonOps dashboard
// offers for commitlog archiving.
var commitLogRemoteTypes = []string{"local", "sftp", "s3", "s3Compatible", "azureblob", "googlecloudstorage"}

var _ resource.Resource = (*cassandraCommitLogSettingsResource)(nil)
var _ resource.ResourceWithImportState = (*cassandraCommitLogSettingsResource)(nil)

type cassandraCommitLogSettingsResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraCommitLogSettingsResource() resource.Resource {
	return &cassandraCommitLogSettingsResource{}
}

func (r *cassandraCommitLogSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *cassandraCommitLogSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_commitlog_settings"
}

func (r *cassandraCommitLogSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Cassandra commitlog archiving for a set of datacenters. AxonOps archives commitlog segments to the configured storage so a cluster can be restored to a point in time. Each datacenter can belong to only one commitlog archive configuration; deleting the resource stops archiving but does not remove commitlogs already archived.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:      true,
				Description:   "The name of the cluster.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cluster_type": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("cassandra"),
				Description:   "The cluster type (cassandra or dse). Default: cassandra",
				Validators:    []validator.String{cassandraOnlyClusterTypeValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"datacenters": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Datacenters whose commitlogs are archived. The first datacenter identifies the configuration; changing the list replaces the resource.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"remote_type": schema.StringAttribute{
				Required:    true,
				Description: "Storage backend: " + strings.Join(commitLogRemoteTypes, ", ") + ".",
				Validators:  []validator.String{stringvalidator.OneOf(commitLogRemoteTypes...)},
			},
			"remote_path": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Path on the storage backend, e.g. a bucket and prefix for s3 or a directory for local and sftp.",
			},
			"remote_retention": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("60d"),
				Description: "How long archived commitlogs are kept. Default: 60d",
				Validators:  []validator.String{durationValidator()},
			},
			"remote_config": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "rclone-style storage configuration as `key = value` lines, e.g. credentials and region for s3. A `type = <remote_type>` line is added when missing.",
			},
			"timeout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("10h"),
				Description: "Upload operation timeout. Default: 10h",
				Validators:  []validator.String{durationValidator()},
			},
			"transfers": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Number of parallel file transfers. 0 uses the agent default. Default: 0",
				Validators:  []validator.Int64{int64validator.AtLeast(0)},
			},
			"bw_limit": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Upload bandwidth limit in rclone format, e.g. \"10M\". Empty means unlimited.",
			},
		},
	}
}

type cassandraCommitLogSettingsResourceData struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	ClusterType     types.String `tfsdk:"cluster_type"`
	Datacenters     types.List   `tfsdk:"datacenters"`
	RemoteType      types.String `tfsdk:"remote_type"`
	RemotePath      types.String `tfsdk:"remote_path"`
	RemoteRetention types.String `tfsdk:"remote_retention"`
	RemoteConfig    types.String `tfsdk:"remote_config"`
	Timeout         types.String `tfsdk:"timeout"`
	Transfers       types.Int64  `tfsdk:"transfers"`
	BwLimit         types.String `tfsdk:"bw_limit"`
}

// commitLogRemoteConfig returns config with a "type = remoteType" line
// prepended when config has no type key, since the agent selects the rclone
// backend from it.
func commitLogRemoteConfig(remoteType, config string) string {
	for _, line := range strings.Split(config, "\n") {
		key, _, found := strings.Cut(line, "=")
		if found && strings.TrimSpace(key) == "type" {
			return config
		}
	}
	typeLine := "type = " + remoteType
	if strings.TrimSpace(config) == "" {
		return typeLine
	}
	return typeLine + "\n" + config
}

func (data *cassandraCommitLogSettingsResourceData) toSettings(ctx context.Context) (axonopsClient.CommitLogArchiveSettings, diag.Diagnostics) {
	var datacenters []string
	diags := data.Datacenters.ElementsAs(ctx, &datacenters, false)
	return axonopsClient.CommitLogArchiveSettings{
		Datacenters:             datacenters,
		RemoteType:              data.RemoteType.ValueString(),
		RemotePath:              data.RemotePath.ValueString(),
		RemoteRetentionDuration: data.RemoteRetention.ValueString(),
		RemoteConfig:            commitLogRemoteConfig(data.RemoteType.ValueString(), data.RemoteConfig.ValueString()),
		Timeout:                 data.Timeout.ValueString(),
		BwLimit:                 data.BwLimit.ValueString(),
		Transfers:               int(data.Transfers.ValueInt64()),
	}, diags
}

// findCommitLogSettings returns the configuration covering datacenter, or nil.
func findCommitLogSettings(all []axonopsClient.CommitLogArchiveSettings, datacenter string) *axonopsClient.CommitLogArchiveSettings {
	for i := range all {
		if slices.Contains(all[i].Datacenters, datacenter) {
			return &all[i]
		}
	}
	return nil
}

// confirmSettings waits until the configuration covering the first
// datacenter reports the values that were sent. remote_config is not
// compared, as the API may not return it as sent.
func (r *cassandraCommitLogSettingsResource) confirmSettings(ctx context.Context, clusterType, clusterName string, want axonopsClient.CommitLogArchiveSettings) error {
	_, err := confirmWrite(ctx, fmt.Sprintf("commitlog archive settings for datacenter %q", want.Datacenters[0]), func(ctx context.Context) (struct{}, bool, error) {
		all, err := r.client.GetCommitLogArchiveSettings(ctx, clusterType, clusterName)
		if err != nil {
			return struct{}{}, false, err
		}
		got := findCommitLogSettings(all, want.Datacenters[0])
		return struct{}{}, got != nil &&
			got.RemoteType == want.RemoteType &&
			got.RemotePath == want.RemotePath &&
			got.RemoteRetentionDuration == want.RemoteRetentionDuration &&
			got.Timeout == want.Timeout, nil
	})
	return err
}

func (r *cassandraCommitLogSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cassandraCommitLogSettingsResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, diags := data.toSettings(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	clusterType, clusterName := data.ClusterType.ValueString(), data.ClusterName.ValueString()

	// A datacenter can belong to one configuration only; refuse to silently
	// take over one that exists outside Terraform.
	existing, err := r.client.GetCommitLogArchiveSettings(ctx, clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read commitlog archive settings: %s", err))
		return
	}
	for _, dc := range settings.Datacenters {
		if findCommitLogSettings(existing, dc) != nil {
			resp.Diagnostics.AddError("Commitlog Archive Already Configured",
				fmt.Sprintf("Datacenter %q of cluster %s/%s already has commitlog archive settings. Import them with ID %s/%s/%s or remove them first.",
					dc, clusterType, clusterName, clusterType, clusterName, dc))
			return
		}
	}

	if err := r.client.CreateCommitLogArchiveSettings(ctx, clusterType, clusterName, settings); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create commitlog archive settings: %s", err))
		return
	}

	if err := r.confirmSettings(ctx, clusterType, clusterName, settings); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm commitlog archive settings were created: %s", err))
		return
	}

	tflog.Info(ctx, "Created Cassandra commitlog archive settings resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraCommitLogSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cassandraCommitLogSettingsResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var datacenters []string
	resp.Diagnostics.Append(data.Datacenters.ElementsAs(ctx, &datacenters, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(datacenters) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	all, err := r.client.GetCommitLogArchiveSettings(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read commitlog archive settings: %s", err))
		return
	}

	found := findCommitLogSettings(all, datacenters[0])
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(setCommitLogSettingsData(ctx, &data, found)...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// setCommitLogSettingsData copies the API values into data. remote_config is
// sensitive and not returned as sent, so the configured value is kept, as
// the backup resource does.
func setCommitLogSettingsData(ctx context.Context, data *cassandraCommitLogSettingsResourceData, s *axonopsClient.CommitLogArchiveSettings) diag.Diagnostics {
	var diags diag.Diagnostics
	data.Datacenters, diags = types.ListValueFrom(ctx, types.StringType, s.Datacenters)
	data.RemoteType = types.StringValue(s.RemoteType)
	data.RemotePath = types.StringValue(s.RemotePath)
	data.RemoteRetention = types.StringValue(s.RemoteRetentionDuration)
	data.Timeout = types.StringValue(s.Timeout)
	data.Transfers = types.Int64Value(int64(s.Transfers))
	data.BwLimit = types.StringValue(s.BwLimit)
	return diags
}

func (r *cassandraCommitLogSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data cassandraCommitLogSettingsResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, diags := data.toSettings(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	clusterType, clusterName := data.ClusterType.ValueString(), data.ClusterName.ValueString()

	if err := r.client.UpdateCommitLogArchiveSettings(ctx, clusterType, clusterName, settings); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update commitlog archive settings: %s", err))
		return
	}

	if err := r.confirmSettings(ctx, clusterType, clusterName, settings); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm commitlog archive settings were updated: %s", err))
		return
	}

	tflog.Info(ctx, "Updated Cassandra commitlog archive settings resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraCommitLogSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cassandraCommitLogSettingsResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var datacenters []string
	resp.Diagnostics.Append(data.Datacenters.ElementsAs(ctx, &datacenters, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCommitLogArchiveSettings(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString(), datacenters)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete commitlog archive settings: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted Cassandra commitlog archive settings resource")
}

// ImportState imports the commitlog archive settings covering a datacenter.
// Import ID format: cluster_type/cluster_name/datacenter
func (r *cassandraCommitLogSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/datacenter, got: %s", req.ID),
		)
		return
	}

	clusterType, clusterName, datacenter := parts[0], parts[1], parts[2]

	all, err := r.client.GetCommitLogArchiveSettings(ctx, clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read commitlog archive settings: %s", err))
		return
	}

	found := findCommitLogSettings(all, datacenter)
	if found == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("No commitlog archive settings for datacenter %s in cluster %s/%s", datacenter, clusterType, clusterName))
		return
	}

	data := cassandraCommitLogSettingsResourceData{
		ClusterName:  types.StringValue(clusterName),
		ClusterType:  types.StringValue(clusterType),
		RemoteConfig: types.StringNull(),
	}
	resp.Diagnostics.Append(setCommitLogSettingsData(ctx, &data, found)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, fmt.Sprintf("Imported commitlog archive settings for datacenter %s in cluster %s/%s", datacenter, clusterType, clusterName))
}
