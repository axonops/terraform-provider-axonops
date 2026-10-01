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

var _ resource.Resource = (*connectorResource)(nil)
var _ resource.ResourceWithImportState = (*connectorResource)(nil)

type connectorResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaConnectConnectorResource() resource.Resource {
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
	resp.TypeName = req.ProviderTypeName + "_kafka_connect_connector"
}

func (r *connectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Kafka Connect connector.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"connect_cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka Connect cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the connector.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.MapAttribute{
				Required:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "The connector configuration as a map of key-value pairs. May contain secrets (e.g. connection credentials).",
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

// refreshConnectorConfig updates the managed config keys with values from the
// API. Only keys already present in managed are tracked, so server-injected
// extra keys (and "name", which Kafka Connect always adds) don't cause
// spurious diffs. A managed key missing from the API response is dropped so
// Terraform plans to set it again.
func refreshConnectorConfig(managed map[string]types.String, remote map[string]string) map[string]types.String {
	if managed == nil {
		return nil
	}
	refreshed := make(map[string]types.String, len(managed))
	for key := range managed {
		if key == "name" {
			continue
		}
		if v, ok := remote[key]; ok {
			refreshed[key] = types.StringValue(v)
		}
	}
	return refreshed
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

	result, err := r.client.CreateConnector(ctx, data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), connector)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create connector, got error: %s", err))
		return
	}

	if err := r.confirmConnector(ctx, data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), data.Name.ValueString(), config); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm connector was created: %s", err))
		return
	}

	// Update computed fields
	data.Type = types.StringValue(result.Type)

	tflog.Info(ctx, "Created connector resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// confirmConnector waits until the connector is listed with every config key
// that was sent at the sent value.
func (r *connectorResource) confirmConnector(ctx context.Context, clusterName, connectCluster, name string, config map[string]string) error {
	_, err := confirmWrite(ctx, fmt.Sprintf("connector %q", name), func(ctx context.Context) (struct{}, bool, error) {
		got, err := r.client.GetConnector(ctx, clusterName, connectCluster, name)
		if err != nil || got == nil {
			return struct{}{}, false, err
		}
		for k, v := range config {
			if got.Config[k] != v {
				return struct{}{}, false, nil
			}
		}
		return struct{}{}, true, nil
	})
	return err
}

func (r *connectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data connectorResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetConnector(ctx, data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read connector, got error: %s", err))
		return
	}

	if result == nil {
		// Connector was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Only refresh config keys already tracked in state; drop keys missing
	// server-side so Terraform re-plans them, and ignore server-injected
	// extra keys (and "name") so they don't cause diffs.
	data.Config = refreshConnectorConfig(data.Config, result.Config)
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

	// Defensive: name change should already be intercepted by
	// RequiresReplace on the "name" attribute, but guard against being
	// called anyway (e.g. programmatic state manipulation).
	if planData.Name.ValueString() != stateData.Name.ValueString() {
		resp.Diagnostics.AddError("Cannot Change Connector Name",
			"Changing the connector name requires destroying and recreating the resource.")
		return
	}

	// Convert config map
	config := make(map[string]string)
	for key, value := range planData.Config {
		config[key] = value.ValueString()
	}

	result, err := r.client.UpdateConnectorConfig(ctx, planData.ClusterName.ValueString(), planData.ConnectClusterName.ValueString(), planData.Name.ValueString(), config)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update connector, got error: %s", err))
		return
	}

	if err := r.confirmConnector(ctx, planData.ClusterName.ValueString(), planData.ConnectClusterName.ValueString(), planData.Name.ValueString(), config); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm connector was updated: %s", err))
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

	err := r.client.DeleteConnector(ctx, data.ClusterName.ValueString(), data.ConnectClusterName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete connector, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted connector resource")
}

// parseConnectorImportID parses a connector import ID of the form
// cluster_name/connect_cluster_name/connector_name. The connector name is
// the most likely field to contain "/", so it absorbs everything after the
// first two fixed fields.
func parseConnectorImportID(id string) (clusterName, connectClusterName, connectorName string, err error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf(
			"expected import ID format: cluster_name/connect_cluster_name/connector_name, got: %s", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// ImportState imports an existing connector into Terraform state.
// Import ID format: cluster_name/connect_cluster_name/connector_name
// The connector name (last field) may itself contain "/".
func (r *connectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	clusterName, connectClusterName, connectorName, err := parseConnectorImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	// Get connector details from the API
	connector, err := r.client.GetConnector(ctx, clusterName, connectClusterName, connectorName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read connector %s: %s", connectorName, err),
		)
		return
	}

	if connector == nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Connector %s not found in cluster %s/%s", connectorName, clusterName, connectClusterName),
		)
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("connect_cluster_name"), connectClusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), connectorName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), connector.Type)...)

	// Filter out "name" key from config
	config := make(map[string]string)
	for key, value := range connector.Config {
		if key == "name" {
			continue
		}
		config[key] = value
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("config"), config)...)

	tflog.Info(ctx, fmt.Sprintf("Imported connector %s from cluster %s/%s", connectorName, clusterName, connectClusterName))
}
