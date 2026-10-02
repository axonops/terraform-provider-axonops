package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Defaults applied by AxonOps when no agent disconnection tolerance is set.
// Delete restores them.
const (
	defaultAgentDisconnectWarnTimeout  = "30s"
	defaultAgentDisconnectErrorTimeout = "1m"
)

var _ resource.Resource = (*cassandraAgentDisconnectToleranceResource)(nil)
var _ resource.ResourceWithImportState = (*cassandraAgentDisconnectToleranceResource)(nil)

type cassandraAgentDisconnectToleranceResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCassandraAgentDisconnectToleranceResource() resource.Resource {
	return &cassandraAgentDisconnectToleranceResource{}
}

func (r *cassandraAgentDisconnectToleranceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *cassandraAgentDisconnectToleranceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cassandra_agent_disconnect_tolerance"
}

func (r *cassandraAgentDisconnectToleranceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages how long the AxonOps agents of a cluster may be disconnected before AxonOps raises a warning and then an error. There is one setting per cluster; deleting the resource restores the AxonOps defaults (30s warning, 1m error).",
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
			"warn_timeout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultAgentDisconnectWarnTimeout),
				Description: "How long an agent may be disconnected before AxonOps raises a warning, e.g. \"30s\" or \"2m\". Default: 30s",
				Validators:  []validator.String{durationValidator()},
			},
			"error_timeout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultAgentDisconnectErrorTimeout),
				Description: "How long an agent may be disconnected before AxonOps raises an error, e.g. \"1m\" or \"5m\". Default: 1m",
				Validators:  []validator.String{durationValidator()},
			},
		},
	}
}

type cassandraAgentDisconnectToleranceResourceData struct {
	ClusterName  types.String `tfsdk:"cluster_name"`
	ClusterType  types.String `tfsdk:"cluster_type"`
	WarnTimeout  types.String `tfsdk:"warn_timeout"`
	ErrorTimeout types.String `tfsdk:"error_timeout"`
}

func (r *cassandraAgentDisconnectToleranceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data cassandraAgentDisconnectToleranceResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set agent disconnection tolerance: %s", err))
		return
	}

	tflog.Info(ctx, "Created Cassandra agent disconnection tolerance resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// apply writes the planned tolerance and waits until reads report it.
func (r *cassandraAgentDisconnectToleranceResource) apply(ctx context.Context, data cassandraAgentDisconnectToleranceResourceData) error {
	clusterType, clusterName := data.ClusterType.ValueString(), data.ClusterName.ValueString()
	want := axonopsClient.AgentDisconnectionTolerance{
		WarnTimeout:  data.WarnTimeout.ValueString(),
		ErrorTimeout: data.ErrorTimeout.ValueString(),
	}

	if err := r.client.UpdateAgentDisconnectionTolerance(ctx, clusterType, clusterName, want); err != nil {
		return err
	}

	_, err := confirmWrite(ctx, "agent disconnection tolerance", func(ctx context.Context) (struct{}, bool, error) {
		got, err := r.client.GetAgentDisconnectionTolerance(ctx, clusterType, clusterName)
		if err != nil || got == nil {
			return struct{}{}, false, err
		}
		return struct{}{}, *withAgentDisconnectDefaults(got) == want, nil
	})
	return err
}

// withAgentDisconnectDefaults fills empty values with the AxonOps defaults,
// since the API returns empty values for a cluster that was never configured.
func withAgentDisconnectDefaults(t *axonopsClient.AgentDisconnectionTolerance) *axonopsClient.AgentDisconnectionTolerance {
	out := *t
	if out.WarnTimeout == "" {
		out.WarnTimeout = defaultAgentDisconnectWarnTimeout
	}
	if out.ErrorTimeout == "" {
		out.ErrorTimeout = defaultAgentDisconnectErrorTimeout
	}
	return &out
}

func (r *cassandraAgentDisconnectToleranceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data cassandraAgentDisconnectToleranceResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.client.GetAgentDisconnectionTolerance(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read agent disconnection tolerance: %s", err))
		return
	}
	got = withAgentDisconnectDefaults(got)

	data.WarnTimeout = types.StringValue(got.WarnTimeout)
	data.ErrorTimeout = types.StringValue(got.ErrorTimeout)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraAgentDisconnectToleranceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data cassandraAgentDisconnectToleranceResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update agent disconnection tolerance: %s", err))
		return
	}

	tflog.Info(ctx, "Updated Cassandra agent disconnection tolerance resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *cassandraAgentDisconnectToleranceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data cassandraAgentDisconnectToleranceResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaults := axonopsClient.AgentDisconnectionTolerance{
		WarnTimeout:  defaultAgentDisconnectWarnTimeout,
		ErrorTimeout: defaultAgentDisconnectErrorTimeout,
	}
	err := r.client.UpdateAgentDisconnectionTolerance(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString(), defaults)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset agent disconnection tolerance: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted (reset) Cassandra agent disconnection tolerance resource")
}

// ImportState imports the agent disconnection tolerance of a cluster.
// Import ID format: cluster_type/cluster_name
func (r *cassandraAgentDisconnectToleranceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]

	got, err := r.client.GetAgentDisconnectionTolerance(ctx, clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read agent disconnection tolerance: %s", err))
		return
	}
	got = withAgentDisconnectDefaults(got)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("warn_timeout"), got.WarnTimeout)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("error_timeout"), got.ErrorTimeout)...)

	tflog.Info(ctx, fmt.Sprintf("Imported agent disconnection tolerance for cluster %s/%s", clusterType, clusterName))
}
