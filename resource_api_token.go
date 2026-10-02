package main

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"time"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// apiTokenRoleRegex matches an AxonOps role scope:
// <org|_global_>/<role>, <org>/<clusterType>/<role> or
// <org>/<clusterType>/<clusterName>/<role>.
var apiTokenRoleRegex = regexp.MustCompile(`^[^/\s]+(/[^/\s]+){0,2}/(superuser|admin|readonly|backupadmin)$`)

var _ resource.Resource = (*apiTokenResource)(nil)
var _ resource.ResourceWithImportState = (*apiTokenResource)(nil)

type apiTokenResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewApiTokenResource() resource.Resource {
	return &apiTokenResource{}
}

func (r *apiTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_token"
}

func (r *apiTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an AxonOps API token for the provider's organisation. Tokens are immutable: changing " +
			"allowed_roles, expires_at or rotation_triggers creates a new token and revokes the old one. " +
			"Requires a superuser API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The token key ID assigned by AxonOps.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "A label for the token. AxonOps does not store token names, so this value is kept " +
					"in Terraform state only and can change without replacing the token.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"allowed_roles": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Role scopes granted to the token. Each entry is `<org>/<role>`, `<org>/<cluster_type>/<role>` " +
					"or `<org>/<cluster_type>/<cluster_name>/<role>`, where role is superuser, admin, readonly or backupadmin.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(apiTokenRoleRegex,
						"must be <org>/<role>, <org>/<cluster_type>/<role> or <org>/<cluster_type>/<cluster_name>/<role> with role superuser, admin, readonly or backupadmin")),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Optional:    true,
				Description: "Expiry time in RFC 3339 format (e.g. 2027-01-01T00:00:00Z). Omit for a token that never expires.",
				Validators:  []validator.String{rfc3339Validator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rotation_triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary values that force a new token when changed. Use it to rotate the token, " +
					"for example with a `time_rotating` resource. Combine with `create_before_destroy`.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"secret": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				Description: "The token secret. AxonOps returns it only when the token is created, " +
					"so it is null for imported tokens. Send it as `Authorization: AxonApi <secret>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation time in RFC 3339 format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

type apiTokenResourceData struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	AllowedRoles     types.List   `tfsdk:"allowed_roles"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
	RotationTriggers types.Map    `tfsdk:"rotation_triggers"`
	Secret           types.String `tfsdk:"secret"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

// expiryEpoch converts expires_at to the API's epoch seconds (0 = never).
func expiryEpoch(v types.String) (int32, error) {
	if v.IsNull() || v.ValueString() == "" {
		return 0, nil
	}
	t, err := time.Parse(time.RFC3339, v.ValueString())
	if err != nil {
		return 0, err
	}
	if t.Unix() <= 0 || t.Unix() > math.MaxInt32 {
		return 0, fmt.Errorf("expires_at %s is out of range", v.ValueString())
	}
	return int32(t.Unix()), nil
}

func epochToRFC3339(s int32) string {
	return time.Unix(int64(s), 0).UTC().Format(time.RFC3339)
}

// applyToken copies the server view of a token into data. expires_at keeps
// the configured spelling when it denotes the same instant.
func applyToken(ctx context.Context, data *apiTokenResourceData, token *axonopsClient.ApiToken) error {
	roles, diags := types.ListValueFrom(ctx, types.StringType, token.AllowedRoles)
	if diags.HasError() {
		return fmt.Errorf("converting allowed_roles: %v", diags)
	}
	data.AllowedRoles = roles
	data.ID = types.StringValue(token.KeyId)

	if token.TokenExpiry == 0 {
		data.ExpiresAt = types.StringNull()
	} else if cur, err := expiryEpoch(data.ExpiresAt); err != nil || cur != token.TokenExpiry {
		data.ExpiresAt = types.StringValue(epochToRFC3339(token.TokenExpiry))
	}

	if token.CreationTime != 0 {
		data.CreatedAt = types.StringValue(epochToRFC3339(token.CreationTime))
	} else if data.CreatedAt.IsUnknown() {
		data.CreatedAt = types.StringNull()
	}
	return nil
}

func (r *apiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data apiTokenResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roles []string
	resp.Diagnostics.Append(data.AllowedRoles.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	expiry, err := expiryEpoch(data.ExpiresAt)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("expires_at"), "Invalid expires_at", err.Error())
		return
	}

	created, err := r.client.CreateApiToken(ctx, roles, expiry)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create API token: %s", err))
		return
	}

	// Save the ID and secret straight away: the secret cannot be fetched
	// again, so it must not be lost if confirmation fails.
	data.ID = types.StringValue(created.ApiKeyId)
	data.Secret = types.StringValue(created.ApiKey)
	data.CreatedAt = types.StringNull()

	token, err := confirmWrite(ctx, fmt.Sprintf("API token %q", created.ApiKeyId), func(ctx context.Context) (*axonopsClient.ApiToken, bool, error) {
		t, err := r.client.GetApiToken(ctx, created.ApiKeyId)
		return t, err == nil && t != nil, err
	})
	if err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm API token was created: %s", err))
		return
	}
	if err := applyToken(ctx, &data, token); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	tflog.Info(ctx, "Created API token", map[string]interface{}{"id": created.ApiKeyId})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *apiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data apiTokenResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.GetApiToken(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read API token: %s", err))
		return
	}
	if token == nil {
		// Revoked outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	if err := applyToken(ctx, &data, token); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update only changes name: every other argument forces replacement.
func (r *apiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state apiTokenResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Name = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data apiTokenResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteApiToken(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to revoke API token: %s", err))
		return
	}
	tflog.Info(ctx, "Revoked API token", map[string]interface{}{"id": data.ID.ValueString()})
}

// ImportState imports a token by key ID. The secret cannot be recovered and
// stays null; name is set from configuration on the next apply.
func (r *apiTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// rfc3339Validator checks that a string is an RFC 3339 timestamp.
type rfc3339Validator struct{}

func (v rfc3339Validator) Description(_ context.Context) string {
	return "value must be an RFC 3339 timestamp, e.g. 2027-01-01T00:00:00Z"
}

func (v rfc3339Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rfc3339Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := expiryEpoch(req.ConfigValue); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid timestamp", fmt.Sprintf("%s: %s", v.Description(ctx), err))
	}
}
