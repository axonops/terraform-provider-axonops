package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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

// dashboardLockKind serializes read-modify-write of a cluster's dashboard
// template, which holds every dashboard of the cluster in one document.
const dashboardLockKind = "dashboardtemplate"

var validDashboardFilterTypes = []string{"query", "custom"}

var _ resource.Resource = (*customDashboardResource)(nil)
var _ resource.ResourceWithImportState = (*customDashboardResource)(nil)

type customDashboardResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewCustomDashboardResource() resource.Resource {
	return &customDashboardResource{}
}

func (r *customDashboardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customDashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_dashboard"
}

func (r *customDashboardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a custom dashboard of an AxonOps cluster. All dashboards of a cluster are stored in one " +
			"template; this resource changes only its own dashboard and keeps the others.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The dashboard UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of cluster (cassandra, kafka or dse).",
				Validators:  []validator.String{clusterTypeValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The dashboard name shown in the AxonOps UI.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"filters": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Dashboard filters (template variables) shown above the panels.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The filter variable name used in panel queries.",
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"label": schema.StringAttribute{
							Optional:    true,
							Description: "The label shown in the UI.",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The filter type: query or custom.",
							Validators:  []validator.String{stringvalidator.OneOf(validDashboardFilterTypes...)},
						},
						"multi": schema.BoolAttribute{
							Optional:    true,
							Description: "Whether more than one value can be selected.",
						},
						"query": schema.StringAttribute{
							Optional:    true,
							Description: "The query that lists filter values (type query).",
						},
						"regex": schema.StringAttribute{
							Optional:    true,
							Description: "A regular expression applied to query results.",
						},
						"values": schema.StringAttribute{
							Optional:    true,
							Description: "Comma-separated filter values (type custom).",
						},
						"custom_logic": schema.StringAttribute{
							Optional:    true,
							Description: "Custom logic applied to the filter.",
						},
						"limit": schema.Int64Attribute{
							Optional:    true,
							Description: "Maximum number of values to list.",
							Validators:  []validator.Int64{int64validator.AtLeast(1)},
						},
					},
				},
			},
			"panels": schema.ListNestedAttribute{
				Required: true,
				Description: "Dashboard panels in display order. A panel of type `row` starts a collapsible group; " +
					"the panels after it belong to that row.",
				Validators: []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed: true,
							Description: "The panel UUID. It is kept by list position, so inserting a panel " +
								"shifts the UUIDs of the panels after it.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"title": schema.StringAttribute{
							Required:    true,
							Description: "The panel title.",
						},
						"type": schema.StringAttribute{
							Required: true,
							Description: "The panel type, e.g. row, line-chart, pie_chart, counter, events_table, " +
								"events_timeline or events_bar_chart.",
							Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"details": schema.StringAttribute{
							Optional: true,
							Description: "Panel settings as a JSON object, e.g. `jsonencode({ queries = [{ query = \"...\", " +
								"legend = \"...\" }] })`. The shape depends on the panel type.",
							Validators: []validator.String{jsonObjectValidator{}},
						},
						"layout": schema.SingleNestedAttribute{
							Required:    true,
							Description: "Panel position and size on the dashboard grid.",
							Attributes: map[string]schema.Attribute{
								"x": schema.Int64Attribute{Required: true, Description: "Column of the top-left corner.", Validators: []validator.Int64{int64validator.AtLeast(0)}},
								"y": schema.Int64Attribute{Required: true, Description: "Row of the top-left corner.", Validators: []validator.Int64{int64validator.AtLeast(0)}},
								"w": schema.Int64Attribute{Required: true, Description: "Width in grid columns.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
								"h": schema.Int64Attribute{Required: true, Description: "Height in grid rows.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
							},
						},
					},
				},
			},
		},
	}
}

type customDashboardData struct {
	ID          types.String           `tfsdk:"id"`
	ClusterType types.String           `tfsdk:"cluster_type"`
	ClusterName types.String           `tfsdk:"cluster_name"`
	Name        types.String           `tfsdk:"name"`
	Filters     []dashboardFilterModel `tfsdk:"filters"`
	Panels      []dashboardPanelModel  `tfsdk:"panels"`
}

type dashboardFilterModel struct {
	Name        types.String `tfsdk:"name"`
	Label       types.String `tfsdk:"label"`
	Type        types.String `tfsdk:"type"`
	Multi       types.Bool   `tfsdk:"multi"`
	Query       types.String `tfsdk:"query"`
	Regex       types.String `tfsdk:"regex"`
	Values      types.String `tfsdk:"values"`
	CustomLogic types.String `tfsdk:"custom_logic"`
	Limit       types.Int64  `tfsdk:"limit"`
}

type dashboardPanelModel struct {
	UUID    types.String         `tfsdk:"uuid"`
	Title   types.String         `tfsdk:"title"`
	Type    types.String         `tfsdk:"type"`
	Details types.String         `tfsdk:"details"`
	Layout  dashboardLayoutModel `tfsdk:"layout"`
}

type dashboardLayoutModel struct {
	X types.Int64 `tfsdk:"x"`
	Y types.Int64 `tfsdk:"y"`
	W types.Int64 `tfsdk:"w"`
	H types.Int64 `tfsdk:"h"`
}

// toAPI builds the API dashboard from data, assigning UUIDs to the dashboard
// and to new panels. It writes the assigned UUIDs back into data.
func (data *customDashboardData) toAPI() axonopsClient.CustomDashboard {
	if data.ID.IsUnknown() || data.ID.ValueString() == "" {
		data.ID = types.StringValue(uuid.NewString())
	}

	dash := axonopsClient.CustomDashboard{
		UUID: data.ID.ValueString(),
		Name: data.Name.ValueString(),
	}

	for _, f := range data.Filters {
		dash.Filters = append(dash.Filters, axonopsClient.DashboardFilter{
			Name:        f.Name.ValueString(),
			Label:       f.Label.ValueString(),
			Type:        f.Type.ValueString(),
			Multi:       f.Multi.ValueBoolPointer(),
			CustomLogic: f.CustomLogic.ValueString(),
			Limit:       int(f.Limit.ValueInt64()),
			Query:       f.Query.ValueString(),
			Regex:       f.Regex.ValueString(),
			Values:      f.Values.ValueStringPointer(),
		})
	}

	for i := range data.Panels {
		p := &data.Panels[i]
		if p.UUID.IsUnknown() || p.UUID.ValueString() == "" {
			p.UUID = types.StringValue(uuid.NewString())
		}
		panel := axonopsClient.CustomPanel{
			UUID:  p.UUID.ValueString(),
			Type:  p.Type.ValueString(),
			Title: p.Title.ValueString(),
			Layout: axonopsClient.PanelLayout{
				X: int(p.Layout.X.ValueInt64()),
				Y: int(p.Layout.Y.ValueInt64()),
				W: int(p.Layout.W.ValueInt64()),
				H: int(p.Layout.H.ValueInt64()),
				I: p.UUID.ValueString(),
			},
		}
		if !p.Details.IsNull() && p.Details.ValueString() != "" {
			panel.Details = json.RawMessage(p.Details.ValueString())
		}
		dash.Panels = append(dash.Panels, panel)
	}
	return dash
}

// fromAPI copies the server view of a dashboard into data. Values that are
// equivalent to what data already holds (JSON details, unset optional
// attributes) keep their current form to avoid spurious diffs.
func (data *customDashboardData) fromAPI(dash *axonopsClient.CustomDashboard) {
	data.ID = types.StringValue(dash.UUID)
	data.Name = types.StringValue(dash.Name)

	prevFilters := data.Filters
	if len(dash.Filters) == 0 {
		if prevFilters != nil {
			data.Filters = []dashboardFilterModel{}
		}
	} else {
		data.Filters = make([]dashboardFilterModel, len(dash.Filters))
		for i, f := range dash.Filters {
			var prev *dashboardFilterModel
			if i < len(prevFilters) {
				prev = &prevFilters[i]
			}
			data.Filters[i] = dashboardFilterModel{
				Name:        types.StringValue(f.Name),
				Type:        types.StringValue(f.Type),
				Label:       optionalString(f.Label, prevFilterField(prev, func(m *dashboardFilterModel) types.String { return m.Label })),
				Query:       optionalString(f.Query, prevFilterField(prev, func(m *dashboardFilterModel) types.String { return m.Query })),
				Regex:       optionalString(f.Regex, prevFilterField(prev, func(m *dashboardFilterModel) types.String { return m.Regex })),
				CustomLogic: optionalString(f.CustomLogic, prevFilterField(prev, func(m *dashboardFilterModel) types.String { return m.CustomLogic })),
				Values:      types.StringPointerValue(f.Values),
				Multi:       types.BoolPointerValue(f.Multi),
				Limit:       types.Int64Null(),
			}
			if f.Limit != 0 {
				data.Filters[i].Limit = types.Int64Value(int64(f.Limit))
			}
		}
	}

	prevPanels := data.Panels
	data.Panels = make([]dashboardPanelModel, len(dash.Panels))
	for i, p := range dash.Panels {
		details := types.StringNull()
		if len(p.Details) > 0 && string(p.Details) != "null" {
			var prev types.String
			if i < len(prevPanels) {
				prev = prevPanels[i].Details
			}
			if !prev.IsNull() && !prev.IsUnknown() && jsonEqual(prev.ValueString(), string(p.Details)) {
				details = prev
			} else {
				var buf bytes.Buffer
				if err := json.Compact(&buf, p.Details); err == nil {
					details = types.StringValue(buf.String())
				} else {
					details = types.StringValue(string(p.Details))
				}
			}
		}
		data.Panels[i] = dashboardPanelModel{
			UUID:    types.StringValue(p.UUID),
			Title:   types.StringValue(p.Title),
			Type:    types.StringValue(p.Type),
			Details: details,
			Layout: dashboardLayoutModel{
				X: types.Int64Value(int64(p.Layout.X)),
				Y: types.Int64Value(int64(p.Layout.Y)),
				W: types.Int64Value(int64(p.Layout.W)),
				H: types.Int64Value(int64(p.Layout.H)),
			},
		}
	}
}

func prevFilterField(prev *dashboardFilterModel, get func(*dashboardFilterModel) types.String) types.String {
	if prev == nil {
		return types.StringNull()
	}
	return get(prev)
}

// optionalString maps an API string to an optional attribute: the API omits
// empty strings, so "" keeps a configured "" and is otherwise null.
func optionalString(v string, prev types.String) types.String {
	if v == "" {
		if !prev.IsNull() && !prev.IsUnknown() && prev.ValueString() == "" {
			return prev
		}
		return types.StringNull()
	}
	return types.StringValue(v)
}

// jsonEqual reports whether two JSON documents are semantically equal.
func jsonEqual(a, b string) bool {
	var va, vb interface{}
	if json.Unmarshal([]byte(a), &va) != nil || json.Unmarshal([]byte(b), &vb) != nil {
		return false
	}
	return reflect.DeepEqual(va, vb)
}

// write applies mutate to the cluster's dashboard template and saves it,
// holding the cluster lock for the whole read-modify-write.
func (r *customDashboardResource) write(ctx context.Context, clusterType, clusterName string, mutate func(*axonopsClient.DashboardTemplate) error) error {
	unlock := lockCluster(dashboardLockKind, clusterType, clusterName)
	defer unlock()

	tmpl, err := r.client.GetDashboardTemplate(ctx, clusterType, clusterName)
	if err != nil {
		return err
	}
	if err := mutate(tmpl); err != nil {
		return err
	}
	return r.client.SetDashboardTemplate(ctx, clusterType, clusterName, *tmpl)
}

// confirm waits until the saved dashboard is visible with the expected name
// and panel count, and returns it.
func (r *customDashboardResource) confirm(ctx context.Context, clusterType, clusterName string, want axonopsClient.CustomDashboard) (*axonopsClient.CustomDashboard, error) {
	return confirmWrite(ctx, fmt.Sprintf("dashboard %q", want.Name), func(ctx context.Context) (*axonopsClient.CustomDashboard, bool, error) {
		tmpl, err := r.client.GetDashboardTemplate(ctx, clusterType, clusterName)
		if err != nil {
			return nil, false, err
		}
		got := axonopsClient.FindCustomDashboard(tmpl, want.UUID)
		return got, got != nil && got.Name == want.Name && len(got.Panels) == len(want.Panels), nil
	})
}

func (r *customDashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data customDashboardData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterType, clusterName := data.ClusterType.ValueString(), data.ClusterName.ValueString()
	dash := data.toAPI()

	err := r.write(ctx, clusterType, clusterName, func(tmpl *axonopsClient.DashboardTemplate) error {
		tmpl.Dashboards = append(tmpl.Dashboards, dash)
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create dashboard: %s", err))
		return
	}

	got, err := r.confirm(ctx, clusterType, clusterName, dash)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm dashboard was created: %s", err))
		return
	}
	data.fromAPI(got)

	tflog.Info(ctx, "Created custom dashboard", map[string]interface{}{"id": dash.UUID})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *customDashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data customDashboardData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tmpl, err := r.client.GetDashboardTemplate(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read dashboard: %s", err))
		return
	}
	dash := axonopsClient.FindCustomDashboard(tmpl, data.ID.ValueString())
	if dash == nil {
		// Deleted outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	data.fromAPI(dash)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *customDashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data customDashboardData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterType, clusterName := data.ClusterType.ValueString(), data.ClusterName.ValueString()
	dash := data.toAPI()

	err := r.write(ctx, clusterType, clusterName, func(tmpl *axonopsClient.DashboardTemplate) error {
		existing := axonopsClient.FindCustomDashboard(tmpl, dash.UUID)
		if existing == nil {
			return fmt.Errorf("dashboard %s no longer exists on cluster %s/%s", dash.UUID, clusterType, clusterName)
		}
		*existing = dash
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dashboard: %s", err))
		return
	}

	got, err := r.confirm(ctx, clusterType, clusterName, dash)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to confirm dashboard was updated: %s", err))
		return
	}
	data.fromAPI(got)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *customDashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data customDashboardData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	err := r.write(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString(), func(tmpl *axonopsClient.DashboardTemplate) error {
		kept := tmpl.Dashboards[:0]
		for _, d := range tmpl.Dashboards {
			if d.UUID != id {
				kept = append(kept, d)
			}
		}
		tmpl.Dashboards = kept
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete dashboard: %s", err))
		return
	}
	tflog.Info(ctx, "Deleted custom dashboard", map[string]interface{}{"id": id})
}

// ImportState imports a dashboard.
// Import ID format: cluster_type/cluster_name/dashboard_uuid
func (r *customDashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/dashboard_uuid, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

// jsonObjectValidator checks that a string is a JSON object.
type jsonObjectValidator struct{}

func (v jsonObjectValidator) Description(_ context.Context) string {
	return "value must be a JSON object"
}

func (v jsonObjectValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonObjectValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &obj); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid JSON", fmt.Sprintf("%s: %s", v.Description(ctx), err))
	}
}
