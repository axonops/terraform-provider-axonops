package main

import (
	"context"
	"fmt"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*aclResource)(nil)
var _ resource.ResourceWithImportState = (*aclResource)(nil)

// validACLResourceTypes lists the resource_type values accepted by the API.
var validACLResourceTypes = []string{"ANY", "TOPIC", "GROUP", "CLUSTER", "TRANSACTIONAL_ID", "DELEGATION_TOKEN", "USER"}

// validACLResourcePatternTypes lists the resource_pattern_type values accepted by the API.
var validACLResourcePatternTypes = []string{"ANY", "MATCH", "LITERAL", "PREFIXED"}

// validACLOperations lists the operation values accepted by the API.
var validACLOperations = []string{"ANY", "ALL", "READ", "WRITE", "CREATE", "DELETE", "ALTER", "DESCRIBE", "CLUSTER_ACTION", "DESCRIBE_CONFIGS", "ALTER_CONFIGS", "IDEMPOTENT_WRITE", "CREATE_TOKENS", "DESCRIBE_TOKENS"}

// validACLPermissionTypes lists the permission_type values accepted by the API.
var validACLPermissionTypes = []string{"ANY", "DENY", "ALLOW"}

type aclResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaACLResource() resource.Resource {
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
	resp.TypeName = req.ProviderTypeName + "_kafka_acl"
}

func (r *aclResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Kafka ACL (Access Control List) entry. All identity fields force replacement on change since ACLs cannot be updated in place.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of resource. Valid values: ANY, TOPIC, GROUP, CLUSTER, TRANSACTIONAL_ID, DELEGATION_TOKEN, USER.",
				Validators: []validator.String{
					stringvalidator.OneOf(validACLResourceTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_pattern_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LITERAL"),
				Description: "The pattern type. Valid values: ANY, MATCH, LITERAL, PREFIXED. Default: LITERAL.",
				Validators: []validator.String{
					stringvalidator.OneOf(validACLResourcePatternTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal": schema.StringAttribute{
				Required:    true,
				Description: "The principal (e.g., User:alice).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"host": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("*"),
				Description: "The host. Default: * (all hosts).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"operation": schema.StringAttribute{
				Required:    true,
				Description: "The operation. Valid values: ANY, ALL, READ, WRITE, CREATE, DELETE, ALTER, DESCRIBE, CLUSTER_ACTION, DESCRIBE_CONFIGS, ALTER_CONFIGS, IDEMPOTENT_WRITE, CREATE_TOKENS, DESCRIBE_TOKENS.",
				Validators: []validator.String{
					stringvalidator.OneOf(validACLOperations...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission_type": schema.StringAttribute{
				Required:    true,
				Description: "The permission type. Valid values: ANY, DENY, ALLOW.",
				Validators: []validator.String{
					stringvalidator.OneOf(validACLPermissionTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

// aclMatches reports whether the given KafkaACL (as returned within an
// ACLResource from GetACLs) matches all identity fields of data. Enum-like
// fields are compared case-insensitively since the API's casing is not
// guaranteed to match what was sent on Create.
func aclMatches(data aclResourceData, res axonopsClient.ACLResource, acl axonopsClient.KafkaACL) bool {
	return strings.EqualFold(res.ResourceType, data.ResourceType.ValueString()) &&
		res.ResourceName == data.ResourceName.ValueString() &&
		strings.EqualFold(res.ResourcePatternType, data.ResourcePatternType.ValueString()) &&
		acl.Principal == data.Principal.ValueString() &&
		acl.Host == data.Host.ValueString() &&
		strings.EqualFold(acl.Operation, data.Operation.ValueString()) &&
		strings.EqualFold(acl.PermissionType, data.PermissionType.ValueString())
}

// findACL searches an ACLResponse for an entry matching all identity fields
// of data, returning true if found.
func findACL(data aclResourceData, resp *axonopsClient.ACLResponse) bool {
	if resp == nil {
		return false
	}
	for _, res := range resp.ACLResources {
		for _, acl := range res.ACLs {
			if aclMatches(data, res, acl) {
				return true
			}
		}
	}
	return false
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

	err := r.client.CreateACL(ctx, data.ClusterName.ValueString(), acl)
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

	aclResponse, err := r.client.GetACLs(ctx, data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACLs, got error: %s", err))
		return
	}

	if !findACL(data, aclResponse) {
		// ACL was deleted outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Update should never be called in practice since all identity fields carry
// RequiresReplace plan modifiers; there is nothing else to change in place.
func (r *aclResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData aclResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

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

	err := r.client.DeleteACL(ctx, data.ClusterName.ValueString(), acl)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ACL, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted ACL resource")
}

// parseACLImportID parses an ACL import ID of the form:
//
//	cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type
//
// Only "principal" is realistically free-text enough to contain "/" (e.g.
// "User:svc/account"), so the ID is split into exactly 8 fields by taking the
// first 4 fields and the last 3 fields as fixed, and joining everything left
// in the middle back into the principal field with "/".
func parseACLImportID(id string) (clusterName, resourceType, resourceName, resourcePatternType, principal, host, operation, permissionType string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) < 8 {
		return "", "", "", "", "", "", "", "", fmt.Errorf(
			"expected import ID format: cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type, got: %s", id)
	}

	clusterName = parts[0]
	resourceType = parts[1]
	resourceName = parts[2]
	resourcePatternType = parts[3]
	host = parts[len(parts)-3]
	operation = parts[len(parts)-2]
	permissionType = parts[len(parts)-1]
	principal = strings.Join(parts[4:len(parts)-3], "/")

	return clusterName, resourceType, resourceName, resourcePatternType, principal, host, operation, permissionType, nil
}

// ImportState imports an existing ACL into Terraform state.
// Import ID format: cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type
//
// The "principal" field is the only identity field likely to contain "/"
// (e.g. "User:svc/account"), so parsing fixes the first 4 fields and the
// last 3 fields and treats anything in between as the (possibly "/"
// containing) principal.
func (r *aclResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	clusterName, resourceType, resourceName, resourcePatternType, principal, host, operation, permissionType, err := parseACLImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

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
