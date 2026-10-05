package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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

// commitLogRemotePathRegex rejects a trailing "/", which the AxonOps server
// strips on write and would otherwise show as a diff on every plan.
var commitLogRemotePathRegex = regexp.MustCompile(`[^/]$`)

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
		Description: "Manages Cassandra commitlog archiving for one datacenter. AxonOps archives commitlog segments to the configured storage so the datacenter can be restored to a point in time. Requires the Cassandra point-in-time restore (PITR) feature on the AxonOps organisation. Deleting the resource stops archiving but does not remove commitlogs already archived.",
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
			"datacenter": schema.StringAttribute{
				Required:      true,
				Description:   "Datacenter whose commitlogs are archived. A datacenter can have one commitlog archive configuration only. Changing it replaces the resource.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"remote_type": schema.StringAttribute{
				Required:    true,
				Description: "Storage backend: " + strings.Join(commitLogRemoteTypes, ", ") + ".",
				Validators:  []validator.String{stringvalidator.OneOf(commitLogRemoteTypes...)},
			},
			"remote_path": schema.StringAttribute{
				Required:    true,
				Description: "Base path on the storage backend, e.g. a bucket and prefix for s3 or a directory for local and sftp. Must not end with \"/\".",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(commitLogRemotePathRegex, `must not end with "/"`),
				},
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
	Datacenter      types.String `tfsdk:"datacenter"`
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

func (data *cassandraCommitLogSettingsResourceData) toSettings() axonopsClient.CommitLogArchiveSettings {
	return axonopsClient.CommitLogArchiveSettings{
		Datacenters:             []string{data.Datacenter.ValueString()},
		RemoteType:              data.RemoteType.ValueString(),
		RemotePath:              data.RemotePath.ValueString(),
		RemoteRetentionDuration: data.RemoteRetention.ValueString(),
		RemoteConfig:            commitLogRemoteConfig(data.RemoteType.ValueString(), data.RemoteConfig.ValueString()),
		Timeout:                 data.Timeout.ValueString(),
		BwLimit:                 data.BwLimit.ValueString(),
		Transfers:               int(data.Transfers.ValueInt64()),
	}
}

// findCommitLogSettings returns the configuration of datacenter, or nil. The
// API keys each configuration by its first (and only) datacenter.
func findCommitLogSettings(all []axonopsClient.CommitLogArchiveSettings, datacenter string) *axonopsClient.CommitLogArchiveSettings {
	for i := range all {
		if len(all[i].Datacenters) > 0 && all[i].Datacenters[0] == datacenter {
			return &all[i]
		}
	}
	return nil
}

// confirmSettings waits until the datacenter's configuration reports the
// values that were sent. remote_config is not compared, as the API hides
// its protected fields.
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
			got.Timeout == want.Timeout &&
			got.Transfers == want.Transfers &&
			got.BwLimit == want.BwLimit, nil
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

	settings := data.toSettings()
	clusterType, clusterName, dc := data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.Datacenter.ValueString()

	// The API refuses a second configuration for a datacenter; check first so
	// the error says how to adopt the existing one.
	existing, err := r.client.GetCommitLogArchiveSettings(ctx, clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read commitlog archive settings: %s", err))
		return
	}
	if findCommitLogSettings(existing, dc) != nil {
		resp.Diagnostics.AddError("Commitlog Archive Already Configured",
			fmt.Sprintf("Datacenter %q of cluster %s/%s already has commitlog archive settings. Import them with ID %s/%s/%s or remove them first.",
				dc, clusterType, clusterName, clusterType, clusterName, dc))
		return
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

	all, err := r.client.GetCommitLogArchiveSettings(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read commitlog archive settings: %s", err))
		return
	}

	found := findCommitLogSettings(all, data.Datacenter.ValueString())
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	setCommitLogSettingsData(&data, found)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// setCommitLogSettingsData copies the API values into data. remote_config is
// sensitive and its protected fields are hidden by the API, so the
// configured value is kept, as the backup resource does.
func setCommitLogSettingsData(data *cassandraCommitLogSettingsResourceData, s *axonopsClient.CommitLogArchiveSettings) {
	data.RemoteType = types.StringValue(s.RemoteType)
	data.RemotePath = types.StringValue(s.RemotePath)
	data.RemoteRetention = types.StringValue(s.RemoteRetentionDuration)
	data.Timeout = types.StringValue(s.Timeout)
	data.Transfers = types.Int64Value(int64(s.Transfers))
	data.BwLimit = types.StringValue(s.BwLimit)
}

func (r *cassandraCommitLogSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data cassandraCommitLogSettingsResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := data.toSettings()
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

	err := r.client.DeleteCommitLogArchiveSettings(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString(), []string{data.Datacenter.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete commitlog archive settings: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted Cassandra commitlog archive settings resource")
}

// ImportState imports the commitlog archive settings of a datacenter.
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
		Datacenter:   types.StringValue(datacenter),
		RemoteConfig: types.StringNull(),
	}
	setCommitLogSettingsData(&data, found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, fmt.Sprintf("Imported commitlog archive settings for datacenter %s in cluster %s/%s", datacenter, clusterType, clusterName))
}
