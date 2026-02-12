package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*topicResource)(nil)
var _ resource.ResourceWithImportState = (*topicResource)(nil)

type topicResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaTopicResource() resource.Resource {
	return &topicResource{}
}

func (r *topicResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

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

func (e *topicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_topic"
}

func (e *topicResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"partitions": schema.Int32Attribute{
				Required: true,
			},
			"replication_factor": schema.Int32Attribute{
				Required: true,
			},
			"cluster_name": schema.StringAttribute{
				Required: true,
			},
			"config": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}

}

type topicResourceData struct {
	Name              types.String            `tfsdk:"name"`
	Partitions        types.Int32             `tfsdk:"partitions"`
	ReplicationFactor types.Int32             `tfsdk:"replication_factor"`
	ClusterName       types.String            `tfsdk:"cluster_name"`
	Config            map[string]types.String `tfsdk:"config"`
}

func (e *topicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data topicResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TF doesn't allow "." in configs so convert from _ to pass through to create function
	var configList []axonopsClient.KafkaTopicConfig
	for key, value := range data.Config {
		configList = append(configList, axonopsClient.KafkaTopicConfig{Name: strings.ReplaceAll(key, "_", "."), Value: value.ValueString()})
	}

	err := e.client.CreateTopic(data.Name.ValueString(), data.ClusterName.ValueString(), data.Partitions.ValueInt32(), data.ReplicationFactor.ValueInt32(), configList)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create topic, got error: %s", err))
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (e *topicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data topicResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read resource using 3rd party API.

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (e *topicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData topicResourceData
	var stateData topicResourceData

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

	if planData.Partitions != stateData.Partitions {
		resp.Diagnostics.AddError("Module Error", fmt.Sprintf("Changing of Partitions not supported yet"))
		return
	}

	if planData.ReplicationFactor != stateData.ReplicationFactor {
		resp.Diagnostics.AddError("Module Error", fmt.Sprintf("Changing of Replication Factor not supported yet"))
		return
	}

	var configList []axonopsClient.KafkaUpdateTopicConfig
	for key, value := range planData.Config {
		configList = append(configList, axonopsClient.KafkaUpdateTopicConfig{Key: strings.ReplaceAll(key, "_", "."), Value: value.ValueString(), Op: "SET"})
	}

	err := e.client.UpdateTopicConfig(planData.Name.ValueString(), planData.ClusterName.ValueString(), planData.Partitions.ValueInt32(), planData.ReplicationFactor.ValueInt32(), configList)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update topic, got error: %s", err))
		return
	}

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (e *topicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data topicResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	err := e.client.DeleteTopic(data.Name.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete topic, got error: %s", err))
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete resource using 3rd party API.
}

// ImportState imports an existing topic into Terraform state.
// Import ID format: cluster_name/topic_name
func (e *topicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID (format: cluster_name/topic_name)
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_name/topic_name, got: %s", req.ID),
		)
		return
	}

	clusterName := parts[0]
	topicName := parts[1]

	// Get topic details from the API
	topic, err := e.client.GetTopic(topicName, clusterName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read topic %s from cluster %s: %s", topicName, clusterName, err),
		)
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), topic.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("partitions"), topic.Partitions)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("replication_factor"), topic.ReplicationFactor)...)

	// Convert config (dots to underscores for Terraform)
	config := make(map[string]string)
	for _, c := range topic.Config {
		key := strings.ReplaceAll(c.Name, ".", "_")
		config[key] = c.Value
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("config"), config)...)

	tflog.Info(ctx, fmt.Sprintf("Imported topic %s from cluster %s", topicName, clusterName))
}
