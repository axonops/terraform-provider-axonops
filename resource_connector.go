package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-kafka-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*connectorResource)(nil)

type connectorResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewConnectorResource() resource.Resource {
	return &connectorResource{}
}

func (r *connectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *connectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connector"
}

func (r *connectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Kafka Connect connector.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"connect_cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka Connect cluster.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the connector.",
			},
			"config": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "The connector configuration as a map of key-value pairs.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the connector (source or sink).",
			},
		},
	}
}

type connectorResourceData struct {
	ClusterName        types.String            `tfsdk:"cluster_name"`
	ConnectClusterName types.String            `tfsdk:"connect_cluster_name"`
	Name               types.String            `tfsdk:"name"`
	Config             map[string]types.String `tfsdk:"config"`
	Type               types.String            `tfsdk:"type"`
}

func (r *connectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data connectorResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert config map
	config := make(map[string]string)
	for key, value := range data.Config {
		config[key] = value.ValueString()
	}

	connector := axonopsClient.KafkaConnector{
		Name:   data.Name.ValueString(),
		Config: config,
	}

	result, err := r.client.CreateConnector(data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), connector)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create connector, got error: %s", err))
		return
	}

	// Update computed fields
	data.Type = types.StringValue(result.Type)

	tflog.Info(ctx, "Created connector resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *connectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data connectorResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetConnector(data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector, got error: %s", err))
		return
	}

	if result == nil {
		// Connector was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with current config from API
	// Filter out "name" key as it's automatically added by Kafka Connect
	config := make(map[string]types.String)
	for key, value := range result.Config {
		if key == "name" {
			continue
		}
		config[key] = types.StringValue(value)
	}
	data.Config = config
	data.Type = types.StringValue(result.Type)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *connectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData connectorResourceData
	var stateData connectorResourceData

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

	// Check if name changed - requires delete and recreate
	if planData.Name.ValueString() != stateData.Name.ValueString() {
		resp.Diagnostics.AddError("Cannot Change Connector Name",
			"Changing the connector name requires destroying and recreating the resource. Use 'terraform taint' or modify lifecycle settings.")
		return
	}

	// Convert config map
	config := make(map[string]string)
	for key, value := range planData.Config {
		config[key] = value.ValueString()
	}

	result, err := r.client.UpdateConnectorConfig(planData.ClusterName.ValueString(), planData.ConnectClusterName.ValueString(), planData.Name.ValueString(), config)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update connector, got error: %s", err))
		return
	}

	// Update computed fields
	planData.Type = types.StringValue(result.Type)

	tflog.Info(ctx, "Updated connector resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *connectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data connectorResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteConnector(data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete connector, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted connector resource")
}
