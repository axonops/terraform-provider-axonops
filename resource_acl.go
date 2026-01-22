package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "axonops-kafka-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*aclResource)(nil)
var _ resource.ResourceWithImportState = (*aclResource)(nil)

type aclResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewACLResource() resource.Resource {
	return &aclResource{}
}

func (r *aclResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *aclResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl"
}

func (r *aclResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Kafka ACL (Access Control List) entry.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"resource_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of resource. Valid values: ANY, TOPIC, GROUP, CLUSTER, TRANSACTIONAL_ID, DELEGATION_TOKEN, USER.",
			},
			"resource_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the resource.",
			},
			"resource_pattern_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LITERAL"),
				Description: "The pattern type. Valid values: ANY, MATCH, LITERAL, PREFIXED. Default: LITERAL.",
			},
			"principal": schema.StringAttribute{
				Required:    true,
				Description: "The principal (e.g., User:alice).",
			},
			"host": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("*"),
				Description: "The host. Default: * (all hosts).",
			},
			"operation": schema.StringAttribute{
				Required:    true,
				Description: "The operation. Valid values: ANY, ALL, READ, WRITE, CREATE, DELETE, ALTER, DESCRIBE, CLUSTER_ACTION, DESCRIBE_CONFIGS, ALTER_CONFIGS, IDEMPOTENT_WRITE, CREATE_TOKENS, DESCRIBE_TOKENS.",
			},
			"permission_type": schema.StringAttribute{
				Required:    true,
				Description: "The permission type. Valid values: ANY, DENY, ALLOW.",
			},
		},
	}
}

type aclResourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	ResourceType        types.String `tfsdk:"resource_type"`
	ResourceName        types.String `tfsdk:"resource_name"`
	ResourcePatternType types.String `tfsdk:"resource_pattern_type"`
	Principal           types.String `tfsdk:"principal"`
	Host                types.String `tfsdk:"host"`
	Operation           types.String `tfsdk:"operation"`
	PermissionType      types.String `tfsdk:"permission_type"`
}

func (r *aclResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data aclResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	acl := axonopsClient.KafkaACL{
		ResourceType:        data.ResourceType.ValueString(),
		ResourceName:        data.ResourceName.ValueString(),
		ResourcePatternType: data.ResourcePatternType.ValueString(),
		Principal:           data.Principal.ValueString(),
		Host:                data.Host.ValueString(),
		Operation:           data.Operation.ValueString(),
		PermissionType:      data.PermissionType.ValueString(),
	}

	err := r.client.CreateACL(data.ClusterName.ValueString(), acl)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ACL, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Created ACL resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *aclResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data aclResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// ACLs don't have a unique identifier for individual reads via API
	// We keep the state as-is since Kafka ACLs are matched by all fields

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *aclResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData aclResourceData
	var stateData aclResourceData

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

	// ACLs cannot be updated in place - delete old and create new
	oldACL := axonopsClient.KafkaACL{
		ResourceType:        stateData.ResourceType.ValueString(),
		ResourceName:        stateData.ResourceName.ValueString(),
		ResourcePatternType: stateData.ResourcePatternType.ValueString(),
		Principal:           stateData.Principal.ValueString(),
		Host:                stateData.Host.ValueString(),
		Operation:           stateData.Operation.ValueString(),
		PermissionType:      stateData.PermissionType.ValueString(),
	}

	err := r.client.DeleteACL(stateData.ClusterName.ValueString(), oldACL)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete old ACL during update, got error: %s", err))
		return
	}

	newACL := axonopsClient.KafkaACL{
		ResourceType:        planData.ResourceType.ValueString(),
		ResourceName:        planData.ResourceName.ValueString(),
		ResourcePatternType: planData.ResourcePatternType.ValueString(),
		Principal:           planData.Principal.ValueString(),
		Host:                planData.Host.ValueString(),
		Operation:           planData.Operation.ValueString(),
		PermissionType:      planData.PermissionType.ValueString(),
	}

	err = r.client.CreateACL(planData.ClusterName.ValueString(), newACL)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create new ACL during update, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Updated ACL resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *aclResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data aclResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	acl := axonopsClient.KafkaACL{
		ResourceType:        data.ResourceType.ValueString(),
		ResourceName:        data.ResourceName.ValueString(),
		ResourcePatternType: data.ResourcePatternType.ValueString(),
		Principal:           data.Principal.ValueString(),
		Host:                data.Host.ValueString(),
		Operation:           data.Operation.ValueString(),
		PermissionType:      data.PermissionType.ValueString(),
	}

	err := r.client.DeleteACL(data.ClusterName.ValueString(), acl)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ACL, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted ACL resource")
}

// ImportState imports an existing ACL into Terraform state.
// Import ID format: cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type
func (r *aclResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID
	parts := strings.Split(req.ID, "/")
	if len(parts) != 8 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type, got: %s", req.ID),
		)
		return
	}

	clusterName := parts[0]
	resourceType := parts[1]
	resourceName := parts[2]
	resourcePatternType := parts[3]
	principal := parts[4]
	host := parts[5]
	operation := parts[6]
	permissionType := parts[7]

	// Set the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_type"), resourceType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_name"), resourceName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_pattern_type"), resourcePatternType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal"), principal)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("host"), host)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("operation"), operation)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_type"), permissionType)...)

	tflog.Info(ctx, fmt.Sprintf("Imported ACL from cluster %s", clusterName))
}
