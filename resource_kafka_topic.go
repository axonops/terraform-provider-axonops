package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*topicResource)(nil)
var _ resource.ResourceWithImportState = (*topicResource)(nil)
var _ resource.ResourceWithModifyPlan = (*topicResource)(nil)

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
		Description: "Manages a Kafka topic. Partitions can only be increased; replication factor changes trigger a reassignment.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Topic name. Changing this forces a new topic.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"partitions": schema.Int32Attribute{
				Required:    true,
				Description: "Number of partitions. Can be increased in place; decreasing is not supported by Kafka.",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"replication_factor": schema.Int32Attribute{
				Required:    true,
				Description: "Replication factor. Changes are applied in place via partition reassignment.",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "Kafka cluster name. Changing this forces a new topic.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Topic configuration. Use underscores instead of dots in keys (e.g. retention_ms).",
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

// ModifyPlan rejects partition decreases at plan time, since Kafka cannot
// reduce the partition count of an existing topic.
func (e *topicResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var planPartitions, statePartitions types.Int32
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("partitions"), &planPartitions)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("partitions"), &statePartitions)...)
	if resp.Diagnostics.HasError() || planPartitions.IsUnknown() || statePartitions.IsNull() {
		return
	}

	if planPartitions.ValueInt32() < statePartitions.ValueInt32() {
		resp.Diagnostics.AddAttributeError(path.Root("partitions"), "Cannot Decrease Partitions",
			fmt.Sprintf("Kafka does not support reducing partitions (current %d, planned %d). Recreate the topic instead.",
				statePartitions.ValueInt32(), planPartitions.ValueInt32()))
	}
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

	err := e.client.CreateTopic(ctx, data.Name.ValueString(), data.ClusterName.ValueString(), data.Partitions.ValueInt32(), data.ReplicationFactor.ValueInt32(), configList)
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

	topic, err := e.client.GetTopic(ctx, data.Name.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read topic, got error: %s", err))
		return
	}
	if topic == nil {
		tflog.Warn(ctx, fmt.Sprintf("Topic %s not found in cluster %s, removing from state", data.Name.ValueString(), data.ClusterName.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	data.Partitions = types.Int32Value(topic.Partitions)
	data.ReplicationFactor = types.Int32Value(topic.ReplicationFactor)
	data.Config = refreshTopicConfig(data.Config, topic.Config)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// refreshTopicConfig updates the managed config keys with values from the API.
// Only keys already in state are tracked, so broker-side explicit configs the
// user never declared do not cause diffs. A managed key missing from the API
// response is dropped so Terraform plans to set it again.
func refreshTopicConfig(managed map[string]types.String, remote []axonopsClient.KafkaTopicConfig) map[string]types.String {
	if managed == nil {
		return nil
	}
	remoteByKey := make(map[string]string, len(remote))
	for _, c := range remote {
		remoteByKey[strings.ReplaceAll(c.Name, ".", "_")] = c.Value
	}
	refreshed := make(map[string]types.String, len(managed))
	for key := range managed {
		if v, ok := remoteByKey[key]; ok {
			refreshed[key] = types.StringValue(v)
		}
	}
	return refreshed
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

	topicName := planData.Name.ValueString()
	clusterName := planData.ClusterName.ValueString()

	if planData.Partitions.ValueInt32() != stateData.Partitions.ValueInt32() {
		if planData.Partitions.ValueInt32() < stateData.Partitions.ValueInt32() {
			resp.Diagnostics.AddError("Cannot Decrease Partitions", "Kafka does not support reducing the partition count of a topic")
			return
		}
		if err := e.client.SetTopicPartitions(ctx, topicName, clusterName, planData.Partitions.ValueInt32()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to increase partitions, got error: %s", err))
			return
		}
	}

	if planData.ReplicationFactor.ValueInt32() != stateData.ReplicationFactor.ValueInt32() {
		if err := e.client.SetTopicReplicationFactor(ctx, topicName, clusterName, planData.ReplicationFactor.ValueInt32()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to change replication factor, got error: %s", err))
			return
		}
	}

	var configList []axonopsClient.KafkaUpdateTopicConfig
	for key, value := range planData.Config {
		configList = append(configList, axonopsClient.KafkaUpdateTopicConfig{Key: strings.ReplaceAll(key, "_", "."), Value: value.ValueString(), Op: "SET"})
	}
	// Reset configs removed from the plan back to the broker default.
	for key := range stateData.Config {
		if _, ok := planData.Config[key]; !ok {
			configList = append(configList, axonopsClient.KafkaUpdateTopicConfig{Key: strings.ReplaceAll(key, "_", "."), Op: "DELETE"})
		}
	}

	if len(configList) > 0 {
		err := e.client.UpdateTopicConfig(ctx, topicName, clusterName, planData.Partitions.ValueInt32(), planData.ReplicationFactor.ValueInt32(), configList)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update topic, got error: %s", err))
			return
		}
	}

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (e *topicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data topicResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := e.client.DeleteTopic(ctx, data.Name.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete topic, got error: %s", err))
		return
	}
}

// ImportState imports an existing topic into Terraform state.
// Import ID format: cluster_name/topic_name
func (e *topicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID (format: cluster_name/topic_name)
	parts := strings.SplitN(req.ID, "/", 2)
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
	topic, err := e.client.GetTopic(ctx, topicName, clusterName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read topic %s from cluster %s: %s", topicName, clusterName, err),
		)
		return
	}
	if topic == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Topic %s not found in cluster %s", topicName, clusterName))
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), topicName)...)
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
