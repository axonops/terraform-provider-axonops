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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*slackIntegrationResource)(nil)
var _ resource.ResourceWithImportState = (*slackIntegrationResource)(nil)

type slackIntegrationResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSlackIntegrationResource() resource.Resource {
	return &slackIntegrationResource{}
}

func (r *slackIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *slackIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_slack_integration"
}

func (r *slackIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Slack integration for AxonOps alerting.",
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
			"webhook_url": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The Slack webhook URL.",
			},
			"channel": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The Slack channel name. Default: empty.",
			},
			"axonops_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "The AxonOps dashboard URL. If empty, the default dashboard URL is used.",
			},
		},
	}
}

type slackIntegrationResourceData struct {
	ID          types.String `tfsdk:"id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	ClusterType types.String `tfsdk:"cluster_type"`
	Name        types.String `tfsdk:"name"`
	WebhookURL  types.String `tfsdk:"webhook_url"`
	Channel     types.String `tfsdk:"channel"`
	AxonopsURL  types.String `tfsdk:"axonops_url"`
}

func (r *slackIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data slackIntegrationResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		Type: "slack",
		Params: map[string]string{
			"name":        data.Name.ValueString(),
			"url":         data.WebhookURL.ValueString(),
			"channel":     data.Channel.ValueString(),
			"axondashUrl": data.AxonopsURL.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Slack integration: %s", err))
		return
	}

	// Read back to get the ID
	integrations, err := r.client.GetIntegrations(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read integrations: %s", err))
		return
	}

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "slack")
	if def == nil {
		resp.Diagnostics.AddError("Client Error", "Integration was created but could not be found")
		return
	}
	data.ID = types.StringValue(def.ID)

	tflog.Info(ctx, "Created Slack integration resource")
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *slackIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data slackIntegrationResourceData
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

	def := axonopsClient.FindIntegrationByNameAndType(integrations, data.Name.ValueString(), "slack")
	if def == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(def.ID)
	// Only update non-sensitive fields from API; sensitive fields (webhook_url)
	// are preserved from state to avoid overwriting with masked values.
	data.Channel = types.StringValue(def.Params["channel"])
	data.AxonopsURL = types.StringValue(def.Params["axondashUrl"])

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *slackIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData slackIntegrationResourceData
	var stateData slackIntegrationResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := axonopsClient.IntegrationPayload{
		ID:   stateData.ID.ValueString(),
		Type: "slack",
		Params: map[string]string{
			"name":        planData.Name.ValueString(),
			"url":         planData.WebhookURL.ValueString(),
			"channel":     planData.Channel.ValueString(),
			"axondashUrl": planData.AxonopsURL.ValueString(),
		},
	}

	err := r.client.CreateOrUpdateIntegration(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Slack integration: %s", err))
		return
	}

	planData.ID = stateData.ID

	tflog.Info(ctx, "Updated Slack integration resource")
	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *slackIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data slackIntegrationResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIntegration(data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Slack integration: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted Slack integration resource")
}

// Import ID format: cluster_type/cluster_name/name
func (r *slackIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	def := axonopsClient.FindIntegrationByNameAndType(integrations, name, "slack")
	if def == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Slack integration '%s' not found", name))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), def.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), def.Params["name"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("webhook_url"), def.Params["url"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("channel"), def.Params["channel"])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("axonops_url"), def.Params["axondashUrl"])...)

	tflog.Info(ctx, fmt.Sprintf("Imported Slack integration '%s' for %s/%s", name, clusterType, clusterName))
}
