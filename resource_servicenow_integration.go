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

var _ resource.Resource = (*servicenowIntegrationResource)(nil)
var _ resource.ResourceWithImportState = (*servicenowIntegrationResource)(nil)

type servicenowIntegrationResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewServiceNowIntegrationResource() resource.Resource {
	return &servicenowIntegrationResource{}
}

func (r *servicenowIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *servicenowIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servicenow_integration"
}

func (r *servicenowIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ServiceNow integration for AxonOps alerting.",
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
			"instance_name": schema.StringAttribute{
				Required:    true,
				Description: "The ServiceNow instance name.",
			},
			"user": schema.StringAttribute{
				Required:    true,
				Description: "The ServiceNow username.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The ServiceNow password.",
			},
		},
	}
}

type servicenowIntegrationResourceData struct {
	ID           types.String `tfsdk:"id"`
	ClusterName  types.String `tfsdk:"cluster_name"`
	ClusterType  types.String `tfsdk:"cluster_type"`
	Name         types.String `tfsdk:"name"`
	InstanceName types.String `tfsdk:"instance_name"`
	User         types.String `tfsdk:"user"`
	Password     types.String `tfsdk:"password"`
}

func (r *servicenowIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data servicenowIntegrationResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		Type: "servicenow",
		Params: map[string]string{
			"name":          data.Name.ValueString(),
			"instance_name": data.InstanceName.ValueString(),
			"user":          data.User.ValueString(),
			"password":      data.Password.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ServiceNow integration: %s", err))
		return
	}

	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read integrations: %s", err))
		return
	}

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "servicenow")
	if def == nil {
		resp.Diagnostics.AddError("Client Error", "Integration was created but could not be found")
		return
	}
	data.ID = types.StringValue(def.ID)

	tflog.Info(ctx, "Created ServiceNow integration resource")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *servicenowIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data servicenowIntegrationResourceData
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

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "servicenow")
	if def == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(def.ID)
	// Update non-sensitive fields from API
	data.InstanceName = types.StringValue(def.Params["instance_name"])
	data.User = types.StringValue(def.Params["user"])
	// Preserve password from state to avoid overwriting with masked values

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *servicenowIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData servicenowIntegrationResourceData
	var stateData servicenowIntegrationResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		ID:   stateData.ID.ValueString(),
		Type: "servicenow",
		Params: map[string]string{
			"name":          planData.Name.ValueString(),
			"instance_name": planData.InstanceName.ValueString(),
			"user":          planData.User.ValueString(),
			"password":      planData.Password.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update ServiceNow integration: %s", err))
		return
	}

	planData.ID = stateData.ID

	tflog.Info(ctx, "Updated ServiceNow integration resource")
	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *servicenowIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data servicenowIntegrationResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ServiceNow integration: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted ServiceNow integration resource")
}

// Import ID format: cluster_type/cluster_name/name
func (r *servicenowIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	def := axonopsClient.FindIntegrationByNameAndType(integrations, name, "servicenow")
	if def == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("ServiceNow integration '%s' not found", name))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), def.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), def.Params["name"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_name"), def.Params["instance_name"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user"), def.Params["user"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), def.Params["password"])...)

	tflog.Info(ctx, fmt.Sprintf("Imported ServiceNow integration '%s' for %s/%s", name, clusterType, clusterName))
}
