package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

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

var _ resource.Resource = (*logCollectorResource)(nil)
var _ resource.ResourceWithImportState = (*logCollectorResource)(nil)

type logCollectorResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewLogCollectorResource() resource.Resource {
	return &logCollectorResource{}
}

func (r *logCollectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *logCollectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logcollector"
}

func (r *logCollectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a log collector configuration for a cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of cluster (e.g., cassandra, kafka, dse).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the log collector.",
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the log collector (auto-generated).",
			},
			"filename": schema.StringAttribute{
				Required:    true,
				Description: "The log file path. Supports Go templating (e.g., {{index . \"comp_jvm_kafka.logs.dir\"}}/server.log).",
			},
			"date_format": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("yyyy-MM-dd HH:mm:ss,SSS"),
				Description: "The date format used in log entries. Default: yyyy-MM-dd HH:mm:ss,SSS",
			},
			"info_regex": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Regex pattern for INFO level log entries.",
			},
			"warning_regex": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Regex pattern for WARNING level log entries.",
			},
			"error_regex": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Regex pattern for ERROR level log entries.",
			},
			"debug_regex": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Regex pattern for DEBUG level log entries.",
			},
			"supported_agent_types": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{types.StringValue("all")})),
				Description: "List of agent types this collector supports (e.g., all, broker, kraft-broker, kraft-controller, zookeeper, schema-registry).",
			},
			"error_alert_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Threshold for error alerts. Default: 0",
			},
			"interval": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("5s"),
				Description: "Interval for log collection. Default: 5s",
			},
			"timeout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1m"),
				Description: "Timeout for log collection. Default: 1m",
			},
			"readonly": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the log collector is read-only. Default: false",
			},
		},
	}
}

type logCollectorResourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	ClusterType         types.String `tfsdk:"cluster_type"`
	Name                types.String `tfsdk:"name"`
	UUID                types.String `tfsdk:"uuid"`
	Filename            types.String `tfsdk:"filename"`
	DateFormat          types.String `tfsdk:"date_format"`
	InfoRegex           types.String `tfsdk:"info_regex"`
	WarningRegex        types.String `tfsdk:"warning_regex"`
	ErrorRegex          types.String `tfsdk:"error_regex"`
	DebugRegex          types.String `tfsdk:"debug_regex"`
	SupportedAgentTypes types.List   `tfsdk:"supported_agent_types"`
	ErrorAlertThreshold types.Int64  `tfsdk:"error_alert_threshold"`
	Interval            types.String `tfsdk:"interval"`
	Timeout             types.String `tfsdk:"timeout"`
	Readonly            types.Bool   `tfsdk:"readonly"`
}

func (r *logCollectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data logCollectorResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get existing log collectors
	existingCollectors, err := r.client.GetLogCollectors(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get existing log collectors, got error: %s", err))
		return
	}

	// Generate a new UUID for this collector
	newUUID := uuid.New().String()

	// Convert supported agent types
	var supportedAgentTypes []string
	diags = data.SupportedAgentTypes.ElementsAs(ctx, &supportedAgentTypes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the new collector config
	newCollector := axonopsClient.LogCollectorConfig{
		Name:                data.Name.ValueString(),
		UUID:                newUUID,
		Filename:            data.Filename.ValueString(),
		DateFormat:          data.DateFormat.ValueString(),
		InfoRegex:           data.InfoRegex.ValueString(),
		WarningRegex:        data.WarningRegex.ValueString(),
		ErrorRegex:          data.ErrorRegex.ValueString(),
		DebugRegex:          data.DebugRegex.ValueString(),
		SupportedAgentType:  supportedAgentTypes,
		ErrorAlertThreshold: int(data.ErrorAlertThreshold.ValueInt64()),
		Interval:            data.Interval.ValueString(),
		Timeout:             data.Timeout.ValueString(),
		Readonly:            data.Readonly.ValueBool(),
	}

	// Add to existing collectors
	allCollectors := append(existingCollectors, newCollector)

	// Update all collectors
	err = r.client.UpdateLogCollectors(data.ClusterType.ValueString(), data.ClusterName.ValueString(), allCollectors)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create log collector, got error: %s", err))
		return
	}

	// Set the UUID in state
	data.UUID = types.StringValue(newUUID)

	tflog.Info(ctx, "Created log collector resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *logCollectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data logCollectorResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get all log collectors
	collectors, err := r.client.GetLogCollectors(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read log collectors, got error: %s", err))
		return
	}

	// Find our collector by filename (primary identifier as per Ansible module)
	var found *axonopsClient.LogCollectorConfig
	for _, c := range collectors {
		if c.Filename == data.Filename.ValueString() {
			found = &c
			break
		}
	}

	// Fallback to name if not found by filename
	if found == nil {
		for _, c := range collectors {
			if c.Name == data.Name.ValueString() {
				found = &c
				break
			}
		}
	}

	if found == nil {
		// Collector was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with current values from API (including potentially new UUID)
	data.UUID = types.StringValue(found.UUID)
	data.Name = types.StringValue(found.Name)
	data.Filename = types.StringValue(found.Filename)
	data.DateFormat = types.StringValue(found.DateFormat)
	data.InfoRegex = types.StringValue(found.InfoRegex)
	data.WarningRegex = types.StringValue(found.WarningRegex)
	data.ErrorRegex = types.StringValue(found.ErrorRegex)
	data.DebugRegex = types.StringValue(found.DebugRegex)
	data.ErrorAlertThreshold = types.Int64Value(int64(found.ErrorAlertThreshold))
	data.Readonly = types.BoolValue(found.Readonly)

	// Handle optional fields - use schema defaults if empty
	if found.Interval != "" {
		data.Interval = types.StringValue(found.Interval)
	} else {
		data.Interval = types.StringValue("5s")
	}
	if found.Timeout != "" {
		data.Timeout = types.StringValue(found.Timeout)
	} else {
		data.Timeout = types.StringValue("1m")
	}

	// Convert supported agent types to list - use schema default if nil
	agentTypes := found.SupportedAgentType
	if agentTypes == nil {
		agentTypes = []string{"all"}
	}
	data.SupportedAgentTypes, diags = types.ListValueFrom(ctx, types.StringType, agentTypes)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *logCollectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData logCollectorResourceData
	var stateData logCollectorResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get existing log collectors
	existingCollectors, err := r.client.GetLogCollectors(planData.ClusterType.ValueString(), planData.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get existing log collectors, got error: %s", err))
		return
	}

	// Convert supported agent types
	var supportedAgentTypes []string
	diags = planData.SupportedAgentTypes.ElementsAs(ctx, &supportedAgentTypes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Find and update our collector by filename (primary identifier)
	found := false
	for i, c := range existingCollectors {
		if c.Filename == stateData.Filename.ValueString() || c.Name == stateData.Name.ValueString() {
			existingCollectors[i] = axonopsClient.LogCollectorConfig{
				Name:                planData.Name.ValueString(),
				UUID:                c.UUID, // Keep the current UUID from API
				Filename:            planData.Filename.ValueString(),
				DateFormat:          planData.DateFormat.ValueString(),
				InfoRegex:           planData.InfoRegex.ValueString(),
				WarningRegex:        planData.WarningRegex.ValueString(),
				ErrorRegex:          planData.ErrorRegex.ValueString(),
				DebugRegex:          planData.DebugRegex.ValueString(),
				SupportedAgentType:  supportedAgentTypes,
				ErrorAlertThreshold: int(planData.ErrorAlertThreshold.ValueInt64()),
				Interval:            planData.Interval.ValueString(),
				Timeout:             planData.Timeout.ValueString(),
				Readonly:            planData.Readonly.ValueBool(),
			}
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Not Found", "Log collector not found in cluster configuration")
		return
	}

	// Update all collectors
	err = r.client.UpdateLogCollectors(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), existingCollectors)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update log collector, got error: %s", err))
		return
	}

	// Keep the UUID from state
	planData.UUID = stateData.UUID

	tflog.Info(ctx, "Updated log collector resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *logCollectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data logCollectorResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get existing log collectors
	existingCollectors, err := r.client.GetLogCollectors(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get existing log collectors, got error: %s", err))
		return
	}

	// Remove our collector from the list (by filename or UUID)
	var updatedCollectors []axonopsClient.LogCollectorConfig
	for _, c := range existingCollectors {
		if c.Filename != data.Filename.ValueString() && c.UUID != data.UUID.ValueString() {
			updatedCollectors = append(updatedCollectors, c)
		}
	}

	// Update all collectors (without our deleted one)
	err = r.client.UpdateLogCollectors(data.ClusterType.ValueString(), data.ClusterName.ValueString(), updatedCollectors)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete log collector, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted log collector resource")
}

// ImportState imports an existing log collector into Terraform state.
// Import ID format: cluster_type/cluster_name/filename
func (r *logCollectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID
	parts := strings.Split(req.ID, "/")
	if len(parts) < 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/filename, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]
	// Filename may contain slashes, so join the rest
	filename := strings.Join(parts[2:], "/")

	// Get all log collectors
	collectors, err := r.client.GetLogCollectors(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read log collectors: %s", err),
		)
		return
	}

	// Find the collector by filename
	var found *axonopsClient.LogCollectorConfig
	for _, c := range collectors {
		if c.Filename == filename {
			found = &c
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Log collector with filename %s not found in cluster %s", filename, clusterName),
		)
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), found.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), found.UUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("filename"), found.Filename)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("date_format"), found.DateFormat)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("info_regex"), found.InfoRegex)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("warning_regex"), found.WarningRegex)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("error_regex"), found.ErrorRegex)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("debug_regex"), found.DebugRegex)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("error_alert_threshold"), int64(found.ErrorAlertThreshold))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("readonly"), found.Readonly)...)

	if found.Interval != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("interval"), found.Interval)...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("interval"), "5s")...)
	}

	if found.Timeout != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timeout"), found.Timeout)...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timeout"), "1m")...)
	}

	agentTypes := found.SupportedAgentType
	if agentTypes == nil {
		agentTypes = []string{"all"}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("supported_agent_types"), agentTypes)...)

	tflog.Info(ctx, fmt.Sprintf("Imported log collector %s from cluster %s/%s", found.Name, clusterType, clusterName))
}
