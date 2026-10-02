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

var _ resource.Resource = (*schemaRegistryCompatibilityResource)(nil)
var _ resource.ResourceWithImportState = (*schemaRegistryCompatibilityResource)(nil)

type schemaRegistryCompatibilityResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSchemaRegistryCompatibilityResource() resource.Resource {
	return &schemaRegistryCompatibilityResource{}
}

func (r *schemaRegistryCompatibilityResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *schemaRegistryCompatibilityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema_registry_compatibility"
}

func (r *schemaRegistryCompatibilityResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the Schema Registry compatibility level of a subject, or the global level when subject is omitted. " +
			"The AxonOps API cannot reset a level to its default, so destroying this resource leaves the last level in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The resource ID: cluster_name/subject, or cluster_name for the global level.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subject": schema.StringAttribute{
				Optional:    true,
				Description: "The subject name (e.g. orders-value). Omit to manage the global compatibility level.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"compatibility_level": schema.StringAttribute{
				Required: true,
				Description: "The compatibility level. Valid values: " +
					strings.Join(axonopsClient.ValidCompatibilityLevels, ", ") + ".",
				Validators: []validator.String{
					stringvalidator.OneOf(axonopsClient.ValidCompatibilityLevels...),
				},
			},
		},
	}
}

type schemaRegistryCompatibilityData struct {
	ID                 types.String `tfsdk:"id"`
	ClusterName        types.String `tfsdk:"cluster_name"`
	Subject            types.String `tfsdk:"subject"`
	CompatibilityLevel types.String `tfsdk:"compatibility_level"`
}

func compatibilityID(clusterName, subject string) string {
	if subject == "" {
		return clusterName
	}
	return clusterName + "/" + subject
}

// apply sets the level and waits until a read returns it.
func (r *schemaRegistryCompatibilityResource) apply(ctx context.Context, data *schemaRegistryCompatibilityData) error {
	cluster := data.ClusterName.ValueString()
	subject := data.Subject.ValueString()
	level := data.CompatibilityLevel.ValueString()

	if err := r.client.SetSchemaCompatibility(ctx, cluster, subject, level); err != nil {
		return err
	}

	_, err := confirmWrite(ctx, fmt.Sprintf("schema registry compatibility %q", compatibilityID(cluster, subject)), func(ctx context.Context) (string, bool, error) {
		got, err := r.client.GetSchemaCompatibility(ctx, cluster, subject)
		return got, err == nil && got == level, err
	})
	if err != nil {
		return err
	}

	data.ID = types.StringValue(compatibilityID(cluster, subject))
	return nil
}

func (r *schemaRegistryCompatibilityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data schemaRegistryCompatibilityData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set schema registry compatibility: %s", err))
		return
	}

	tflog.Info(ctx, "Created schema registry compatibility resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *schemaRegistryCompatibilityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data schemaRegistryCompatibilityData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	level, err := r.client.GetSchemaCompatibility(ctx, data.ClusterName.ValueString(), data.Subject.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schema registry compatibility: %s", err))
		return
	}

	if level == "" {
		// The subject-level setting was removed outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	data.CompatibilityLevel = types.StringValue(level)
	data.ID = types.StringValue(compatibilityID(data.ClusterName.ValueString(), data.Subject.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *schemaRegistryCompatibilityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data schemaRegistryCompatibilityData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update schema registry compatibility: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete only removes the resource from state: the AxonOps API has no
// endpoint to reset a compatibility level to its default.
func (r *schemaRegistryCompatibilityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data schemaRegistryCompatibilityData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning(
		"Compatibility level left in place",
		fmt.Sprintf("The AxonOps API cannot reset a Schema Registry compatibility level. %q keeps level %s; it is only removed from Terraform state.",
			data.ID.ValueString(), data.CompatibilityLevel.ValueString()),
	)
}

// ImportState imports a compatibility level.
// Import ID format: cluster_name (global level) or cluster_name/subject.
func (r *schemaRegistryCompatibilityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	cluster, subject, hasSubject := strings.Cut(req.ID, "/")
	if cluster == "" || (hasSubject && subject == "") {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_name or cluster_name/subject, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), cluster)...)
	if hasSubject {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subject"), subject)...)
	}
}
