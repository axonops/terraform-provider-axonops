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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// validSchemaTypes lists the schema_type values accepted by the Schema Registry.
var validSchemaTypes = []string{"AVRO", "JSON", "PROTOBUF"}

var _ resource.Resource = (*schemaResource)(nil)
var _ resource.ResourceWithImportState = (*schemaResource)(nil)

type schemaResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSchemaResource() resource.Resource {
	return &schemaResource{}
}

func (r *schemaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *schemaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema"
}

func (r *schemaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Schema Registry schema subject.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subject": schema.StringAttribute{
				Required:    true,
				Description: "The subject name (e.g., topic-name-value or topic-name-key).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"schema": schema.StringAttribute{
				Required:    true,
				Description: "The schema definition (JSON string for AVRO/JSON, proto definition for PROTOBUF).",
			},
			"schema_type": schema.StringAttribute{
				Required:    true,
				Description: "The schema type. Valid values: AVRO, PROTOBUF, JSON.",
				Validators: []validator.String{
					stringvalidator.OneOf(validSchemaTypes...),
				},
			},
			"schema_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The unique ID assigned to the schema by the Schema Registry.",
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "The version number of the schema.",
			},
		},
	}
}

type schemaResourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	Subject     types.String `tfsdk:"subject"`
	Schema      types.String `tfsdk:"schema"`
	SchemaType  types.String `tfsdk:"schema_type"`
	SchemaId    types.Int64  `tfsdk:"schema_id"`
	Version     types.Int64  `tfsdk:"version"`
}

func (r *schemaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data schemaResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	schemaReq := axonopsClient.CreateSchemaRequest{
		Schema:     data.Schema.ValueString(),
		SchemaType: data.SchemaType.ValueString(),
	}

	result, err := r.client.CreateSchema(ctx, data.ClusterName.ValueString(), data.Subject.ValueString(), schemaReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create schema, got error: %s", err))
		return
	}

	// Set the schema ID from the response
	data.SchemaId = types.Int64Value(int64(result.Id))

	// Confirm the new schema is the latest version and capture it.
	schemaInfo, err := r.confirmSchema(ctx, data.ClusterName.ValueString(), data.Subject.ValueString(), result.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm schema was created: %s", err))
		return
	}
	data.Version = types.Int64Value(int64(schemaInfo.Version))

	tflog.Info(ctx, "Created schema resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// confirmSchema waits until the latest version of subject is the schema with
// id returned by the create call.
func (r *schemaResource) confirmSchema(ctx context.Context, clusterName, subject string, id int) (*axonopsClient.SchemaRegistryVersionedSchema, error) {
	return confirmWrite(ctx, fmt.Sprintf("schema subject %q", subject), func(ctx context.Context) (*axonopsClient.SchemaRegistryVersionedSchema, bool, error) {
		info, err := r.client.GetSchema(ctx, clusterName, subject, "latest")
		if err != nil || info == nil {
			return nil, false, err
		}
		return info, info.Id == id, nil
	})
}

func (r *schemaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data schemaResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetSchema(ctx, data.ClusterName.ValueString(), data.Subject.ValueString(), "latest")
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schema, got error: %s", err))
		return
	}

	if result == nil {
		// Schema was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Only update computed fields from API
	// Don't update Schema or SchemaType as the API returns minified JSON
	// which would cause unnecessary diffs with formatted Terraform config
	data.SchemaId = types.Int64Value(int64(result.Id))
	data.Version = types.Int64Value(int64(result.Version))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *schemaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData schemaResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Schema Registry allows posting new versions to the same subject
	// This creates a new version of the schema
	schemaReq := axonopsClient.CreateSchemaRequest{
		Schema:     planData.Schema.ValueString(),
		SchemaType: planData.SchemaType.ValueString(),
	}

	result, err := r.client.CreateSchema(ctx, planData.ClusterName.ValueString(), planData.Subject.ValueString(), schemaReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update schema, got error: %s", err))
		return
	}

	// Set the new schema ID
	planData.SchemaId = types.Int64Value(int64(result.Id))

	// Confirm the new schema is the latest version and capture it.
	schemaInfo, err := r.confirmSchema(ctx, planData.ClusterName.ValueString(), planData.Subject.ValueString(), result.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm schema was updated: %s", err))
		return
	}
	planData.Version = types.Int64Value(int64(schemaInfo.Version))

	tflog.Info(ctx, "Updated schema resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *schemaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data schemaResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSchema(ctx, data.ClusterName.ValueString(), data.Subject.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete schema, got error: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted schema resource")
}

// ImportState imports an existing schema into Terraform state.
// Import ID format: cluster_name/subject. Subject may itself contain "/", so
// the ID is split into exactly 2 fields with SplitN, letting the subject
// absorb everything after the first "/".
func (r *schemaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_name/subject, got: %s", req.ID),
		)
		return
	}

	clusterName := parts[0]
	subject := parts[1]

	// Get schema details from the API
	schemaInfo, err := r.client.GetSchema(ctx, clusterName, subject, "latest")
	if err != nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Unable to read schema %s: %s", subject, err),
		)
		return
	}

	if schemaInfo == nil {
		resp.Diagnostics.AddError(
			"Import Error",
			fmt.Sprintf("Schema subject %s not found in cluster %s", subject, clusterName),
		)
		return
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subject"), subject)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schema"), schemaInfo.Schema)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schema_type"), schemaInfo.Type)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schema_id"), int64(schemaInfo.Id))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), int64(schemaInfo.Version))...)

	tflog.Info(ctx, fmt.Sprintf("Imported schema %s from cluster %s", subject, clusterName))
}
