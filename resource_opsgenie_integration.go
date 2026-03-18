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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*opsgenieIntegrationResource)(nil)
var _ resource.ResourceWithImportState = (*opsgenieIntegrationResource)(nil)

type opsgenieIntegrationResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewOpsGenieIntegrationResource() resource.Resource {
	return &opsgenieIntegrationResource{}
}

func (r *opsgenieIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *opsgenieIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_opsgenie_integration"
}

func (r *opsgenieIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpsGenie integration for AxonOps alerting.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The integration ID.",
			},
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the integration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"opsgenie_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The OpsGenie API key.",
			},
		},
	}
}

type opsgenieIntegrationResourceData struct {
	ID          types.String `tfsdk:"id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	ClusterType types.String `tfsdk:"cluster_type"`
	Name        types.String `tfsdk:"name"`
	OpsgenieKey types.String `tfsdk:"opsgenie_key"`
}

func (r *opsgenieIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data opsgenieIntegrationResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		Type: "opsgenie",
		Params: map[string]string{
			"name":         data.Name.ValueString(),
			"opsgenie_key": data.OpsgenieKey.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create OpsGenie integration: %s", err))
		return
	}

	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read integrations: %s", err))
		return
	}

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "opsgenie")
	if def == nil {
		resp.Diagnostics.AddError("Client Error", "Integration was created but could not be found")
		return
	}
	data.ID = types.StringValue(def.ID)

	tflog.Info(ctx, "Created OpsGenie integration resource")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *opsgenieIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data opsgenieIntegrationResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "opsgenie")
	if def == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(def.ID)
	// Preserve opsgenie_key from state to avoid overwriting with masked values

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *opsgenieIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData opsgenieIntegrationResourceData
	var stateData opsgenieIntegrationResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		ID:   stateData.ID.ValueString(),
		Type: "opsgenie",
		Params: map[string]string{
			"name":         planData.Name.ValueString(),
			"opsgenie_key": planData.OpsgenieKey.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update OpsGenie integration: %s", err))
		return
	}

	planData.ID = stateData.ID

	tflog.Info(ctx, "Updated OpsGenie integration resource")
	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *opsgenieIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data opsgenieIntegrationResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete OpsGenie integration: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted OpsGenie integration resource")
}

// Import ID format: cluster_type/cluster_name/name
func (r *opsgenieIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Expected format: cluster_type/cluster_name/name, got: %s", req.ID))
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]
	name := parts[2]

	integrations, err := r.client.GetIntegrations(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to get integrations: %s", err))
		return
	}

	def := axonopsClient.FindIntegrationByNameAndType(integrations, name, "opsgenie")
	if def == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("OpsGenie integration '%s' not found", name))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), def.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), def.Params["name"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("opsgenie_key"), def.Params["opsgenie_key"])...)

	tflog.Info(ctx, fmt.Sprintf("Imported OpsGenie integration '%s' for %s/%s", name, clusterType, clusterName))
}
