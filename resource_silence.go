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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*silenceResource)(nil)
var _ resource.ResourceWithImportState = (*silenceResource)(nil)

type silenceResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSilenceResource() resource.Resource {
	return &silenceResource{}
}

func (r *silenceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *silenceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_silence"
}

func (r *silenceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a silence window for a cluster. Silences suppress alerts during maintenance or other planned activities.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of cluster (e.g., cassandra, kafka).",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the silence window.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the silence is active. Default: true",
			},
			"is_recurring": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the silence is recurring based on the cron expression. Default: false",
			},
			"cron_expr": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0 * * * *"),
				Description: "Cron expression for recurring silences. Also used as a unique identifier for the silence. Default: '0 * * * *'",
			},
			"duration": schema.StringAttribute{
				Required:    true,
				Description: "Duration of the silence (e.g., '1h', '30m', '2h30m').",
			},
			"datacenters": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "List of datacenters or nodes to apply the silence to. Empty means all.",
			},
		},
	}
}

type silenceResourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	ClusterType types.String `tfsdk:"cluster_type"`
	ID          types.String `tfsdk:"id"`
	Active      types.Bool   `tfsdk:"active"`
	IsRecurring types.Bool   `tfsdk:"is_recurring"`
	CronExpr    types.String `tfsdk:"cron_expr"`
	Duration    types.String `tfsdk:"duration"`
	Datacenters types.List   `tfsdk:"datacenters"`
}

func (r *silenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data silenceResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var datacenters []string
	diags = data.Datacenters.ElementsAs(ctx, &datacenters, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if datacenters == nil {
		datacenters = []string{}
	}

	silenceID := uuid.New().String()

	silence := axonopsClient.SilenceWindow{
		ID:          silenceID,
		Active:      data.Active.ValueBool(),
		CronExpr:    data.CronExpr.ValueString(),
		IsRecurring: data.IsRecurring.ValueBool(),
		Duration:    data.Duration.ValueString(),
		DCs:         datacenters,
	}

	err := r.client.CreateSilenceWindow(data.ClusterType.ValueString(), data.ClusterName.ValueString(), silence)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create silence: %s", err))
		return
	}

	// Fetch the created silence to confirm and get the actual ID
	silences, err := r.client.GetSilenceWindows(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read silences after creation: %s", err))
		return
	}

	// Find by cron expression (used as identifier per Ansible module logic)
	found := axonopsClient.FindSilenceWindowByCronExpr(silences, data.CronExpr.ValueString())
	if found != nil {
		data.ID = types.StringValue(found.ID)
	} else {
		// Fallback to the ID we generated
		data.ID = types.StringValue(silenceID)
	}

	tflog.Info(ctx, "Created silence resource", map[string]any{
		"cluster_name": data.ClusterName.ValueString(),
		"cluster_type": data.ClusterType.ValueString(),
		"id":           data.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *silenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data silenceResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	silences, err := r.client.GetSilenceWindows(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read silences: %s", err))
		return
	}

	// Try to find by ID first, then by cron expression
	var found *axonopsClient.SilenceWindow
	if data.ID.ValueString() != "" {
		found = axonopsClient.FindSilenceWindowByID(silences, data.ID.ValueString())
	}
	if found == nil {
		found = axonopsClient.FindSilenceWindowByCronExpr(silences, data.CronExpr.ValueString())
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(found.ID)
	data.Active = types.BoolValue(found.Active)
	data.CronExpr = types.StringValue(found.CronExpr)
	data.IsRecurring = types.BoolValue(found.IsRecurring)
	data.Duration = types.StringValue(found.Duration)

	dcs := found.DCs
	if dcs == nil {
		dcs = []string{}
	}
	data.Datacenters, diags = types.ListValueFrom(ctx, types.StringType, dcs)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *silenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data silenceResourceData
	var state silenceResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing silence if we have an ID
	if state.ID.ValueString() != "" {
		err := r.client.DeleteSilenceWindow(state.ClusterType.ValueString(), state.ClusterName.ValueString(), state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete existing silence during update: %s", err))
			return
		}
	}

	var datacenters []string
	diags = data.Datacenters.ElementsAs(ctx, &datacenters, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if datacenters == nil {
		datacenters = []string{}
	}

	silenceID := uuid.New().String()

	silence := axonopsClient.SilenceWindow{
		ID:          silenceID,
		Active:      data.Active.ValueBool(),
		CronExpr:    data.CronExpr.ValueString(),
		IsRecurring: data.IsRecurring.ValueBool(),
		Duration:    data.Duration.ValueString(),
		DCs:         datacenters,
	}

	err := r.client.CreateSilenceWindow(data.ClusterType.ValueString(), data.ClusterName.ValueString(), silence)
	if err != nil {
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Deleted existing silence but failed to create replacement: %s", err))
		return
	}

	// Fetch the created silence to get its actual ID
	silences, err := r.client.GetSilenceWindows(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read silences after update: %s", err))
		return
	}

	found := axonopsClient.FindSilenceWindowByCronExpr(silences, data.CronExpr.ValueString())
	if found != nil {
		data.ID = types.StringValue(found.ID)
	} else {
		data.ID = types.StringValue(silenceID)
	}

	tflog.Info(ctx, "Updated silence resource", map[string]any{
		"cluster_name": data.ClusterName.ValueString(),
		"cluster_type": data.ClusterType.ValueString(),
		"id":           data.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *silenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data silenceResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	silenceID := data.ID.ValueString()
	if silenceID == "" {
		// Try to find by cron expression
		silences, err := r.client.GetSilenceWindows(data.ClusterType.ValueString(), data.ClusterName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read silences: %s", err))
			return
		}

		found := axonopsClient.FindSilenceWindowByCronExpr(silences, data.CronExpr.ValueString())
		if found != nil {
			silenceID = found.ID
		}
	}

	if silenceID != "" {
		err := r.client.DeleteSilenceWindow(data.ClusterType.ValueString(), data.ClusterName.ValueString(), silenceID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete silence: %s", err))
			return
		}
	}

	tflog.Info(ctx, "Deleted silence resource", map[string]any{
		"cluster_name": data.ClusterName.ValueString(),
		"cluster_type": data.ClusterType.ValueString(),
		"id":           silenceID,
	})
}

// ImportState imports an existing silence.
// Import ID format: cluster_type/cluster_name/silence_id
func (r *silenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/silence_id, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]
	silenceID := parts[2]

	silences, err := r.client.GetSilenceWindows(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read silences: %s", err))
		return
	}

	found := axonopsClient.FindSilenceWindowByID(silences, silenceID)
	if found == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("No silence found with ID '%s' in cluster '%s'", silenceID, clusterName))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("active"), found.Active)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_recurring"), found.IsRecurring)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cron_expr"), found.CronExpr)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("duration"), found.Duration)...)

	dcs := found.DCs
	if dcs == nil {
		dcs = []string{}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("datacenters"), dcs)...)

	tflog.Info(ctx, "Imported silence", map[string]any{
		"cluster_type": clusterType,
		"cluster_name": clusterName,
		"id":           silenceID,
	})
}
