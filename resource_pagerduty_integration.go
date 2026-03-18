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

var _ resource.Resource = (*pagerdutyIntegrationResource)(nil)
var _ resource.ResourceWithImportState = (*pagerdutyIntegrationResource)(nil)

type pagerdutyIntegrationResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewPagerDutyIntegrationResource() resource.Resource {
	return &pagerdutyIntegrationResource{}
}

func (r *pagerdutyIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *pagerdutyIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pagerduty_integration"
}

func (r *pagerdutyIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PagerDuty integration for AxonOps alerting.",
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
			"integration_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The PagerDuty integration key.",
			},
		},
	}
}

type pagerdutyIntegrationResourceData struct {
	ID             types.String `tfsdk:"id"`
	ClusterName    types.String `tfsdk:"cluster_name"`
	ClusterType    types.String `tfsdk:"cluster_type"`
	Name           types.String `tfsdk:"name"`
	IntegrationKey types.String `tfsdk:"integration_key"`
}

func (r *pagerdutyIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data pagerdutyIntegrationResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		Type: "pagerduty",
		Params: map[string]string{
			"name":            data.Name.ValueString(),
			"integration_key": data.IntegrationKey.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create PagerDuty integration: %s", err))
		return
	}

	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read integrations: %s", err))
		return
	}

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "pagerduty")
	if def == nil {
		resp.Diagnostics.AddError("Client Error", "Integration was created but could not be found")
		return
	}
	data.ID = types.StringValue(def.ID)

	tflog.Info(ctx, "Created PagerDuty integration resource")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *pagerdutyIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pagerdutyIntegrationResourceData
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

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "pagerduty")
	if def == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(def.ID)
	// Preserve integration_key from state to avoid overwriting with masked values

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *pagerdutyIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData pagerdutyIntegrationResourceData
	var stateData pagerdutyIntegrationResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		ID:   stateData.ID.ValueString(),
		Type: "pagerduty",
		Params: map[string]string{
			"name":            planData.Name.ValueString(),
			"integration_key": planData.IntegrationKey.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update PagerDuty integration: %s", err))
		return
	}

	planData.ID = stateData.ID

	tflog.Info(ctx, "Updated PagerDuty integration resource")
	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *pagerdutyIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pagerdutyIntegrationResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete PagerDuty integration: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted PagerDuty integration resource")
}

// Import ID format: cluster_type/cluster_name/name
func (r *pagerdutyIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	def := axonopsClient.FindIntegrationByNameAndType(integrations, name, "pagerduty")
	if def == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("PagerDuty integration '%s' not found", name))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), def.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), def.Params["name"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integration_key"), def.Params["integration_key"])...)

	tflog.Info(ctx, fmt.Sprintf("Imported PagerDuty integration '%s' for %s/%s", name, clusterType, clusterName))
}
