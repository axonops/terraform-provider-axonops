package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// simpleQueryStringOperators are simple_query_string special characters. If any
// appear in content, the user is assumed to know the query DSL and the value
// is passed through unchanged.
const simpleQueryStringOperators = `+-|*()~"`

// normaliseLogContent rewrites a multi-word content value so Elasticsearch's
// simple_query_string performs an AND match across every term. Single words,
// empty values, and strings already containing query operators are returned
// unchanged.
func normaliseLogContent(content string) string {
	if content == "" {
		return content
	}
	if strings.ContainsAny(content, simpleQueryStringOperators) {
		return content
	}
	parts := strings.Fields(content)
	if len(parts) <= 1 {
		return content
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = "+" + p
	}
	return strings.Join(out, " ")
}

var _ resource.Resource = (*logAlertRuleResource)(nil)
var _ resource.ResourceWithImportState = (*logAlertRuleResource)(nil)

type logAlertRuleResource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewLogAlertRuleResource() resource.Resource {
	return &logAlertRuleResource{}
}

func (r *logAlertRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *logAlertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_log_alert_rule"
}

func (r *logAlertRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a log alert rule for a cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the log alert rule (auto-generated).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the log alert rule.",
			},
			"content": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Description: "The log content/phrase to search for. Multi-word values are " +
					"automatically rewritten as a simple_query_string AND match (e.g. " +
					"`is now DOWN` is sent as `+is +now +DOWN`). To opt out, include any " +
					"simple_query_string operator (`+`, `-`, `|`, `*`, `(`, `)`, `~`, `\"`) " +
					"and the value is passed through unchanged.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Description of the log alert rule.",
			},
			"operator": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(">="),
				Description: "Comparison operator: >, >=, =, !=, <=, <. Defaults to >=.",
			},
			"warning_value": schema.Float64Attribute{
				Required:    true,
				Description: "Warning threshold value.",
			},
			"critical_value": schema.Float64Attribute{
				Required:    true,
				Description: "Critical threshold value.",
			},
			"duration": schema.StringAttribute{
				Required:    true,
				Description: "Duration/time window for log scraping (e.g., 5m, 1h, 24h).",
			},
			"level": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Log level filter (debug, error, warning, info). Comma-separated for multiple.",
			},
			"log_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Log type filter. Comma-separated for multiple.",
			},
			"source": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Log source path filter. Comma-separated for multiple.",
			},
			"present": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the alert rule is active. Defaults to true.",
			},
		},
	}
}

type logAlertRuleResourceData struct {
	ClusterName   types.String  `tfsdk:"cluster_name"`
	ClusterType   types.String  `tfsdk:"cluster_type"`
	ID            types.String  `tfsdk:"id"`
	Name          types.String  `tfsdk:"name"`
	Content       types.String  `tfsdk:"content"`
	Description   types.String  `tfsdk:"description"`
	Operator      types.String  `tfsdk:"operator"`
	WarningValue  types.Float64 `tfsdk:"warning_value"`
	CriticalValue types.Float64 `tfsdk:"critical_value"`
	Duration      types.String  `tfsdk:"duration"`
	Level         types.String  `tfsdk:"level"`
	LogType       types.String  `tfsdk:"log_type"`
	Source        types.String  `tfsdk:"source"`
	Present       types.Bool    `tfsdk:"present"`
}

// buildEventsExpr constructs the events{...} expression from individual fields.
func buildEventsExpr(content, level, source, logType string) string {
	var parts []string
	if content != "" {
		// Escape embedded double quotes so the expression parser doesn't break
		escaped := strings.ReplaceAll(content, `"`, `\"`)
		parts = append(parts, fmt.Sprintf("message=\"%s\"", escaped))
	}
	if level != "" {
		parts = append(parts, fmt.Sprintf("level=\"%s\"", strings.ReplaceAll(level, ",", "|")))
	}
	if source != "" {
		parts = append(parts, fmt.Sprintf("source=\"%s\"", strings.ReplaceAll(source, ",", "|")))
	}
	if logType != "" {
		parts = append(parts, fmt.Sprintf("type=\"%s\"", strings.ReplaceAll(logType, ",", "|")))
	}
	return fmt.Sprintf("events{%s}", strings.Join(parts, ","))
}

// parseEventsExpr parses the events{...} expression back into individual fields.
func parseEventsExpr(expr string) (content, level, source, logType string) {
	pattern := regexp.MustCompile(`events\{(.+)\}`)
	match := pattern.FindStringSubmatch(expr)
	if match == nil {
		return
	}

	elemPattern := regexp.MustCompile(`(\w+)="((?:[^"\\]|\\.)*?)"`)
	elements := elemPattern.FindAllStringSubmatch(match[1], -1)
	for _, elem := range elements {
		key, value := elem[1], elem[2]
		// Unescape escaped double quotes
		value = strings.ReplaceAll(value, `\"`, `"`)
		value = strings.ReplaceAll(value, "|", ",")
		switch key {
		case "message":
			content = value
		case "level":
			level = value
		case "source":
			source = value
		case "type":
			logType = value
		}
	}
	return
}

// isLogAlertRule checks if a MetricAlertRule is actually a log alert rule.
func isLogAlertRule(rule axonopsClient.MetricAlertRule) bool {
	return strings.Contains(rule.Expr, "events{")
}

func (r *logAlertRuleResource) buildRule(data *logAlertRuleResourceData) axonopsClient.MetricAlertRule {
	eventsExpr := buildEventsExpr(
		normaliseLogContent(data.Content.ValueString()),
		data.Level.ValueString(),
		data.Source.ValueString(),
		data.LogType.ValueString(),
	)

	return axonopsClient.MetricAlertRule{
		ID:            data.ID.ValueString(),
		Alert:         data.Name.ValueString(),
		For:           data.Duration.ValueString(),
		Operator:      data.Operator.ValueString(),
		WarningValue:  data.WarningValue.ValueFloat64(),
		CriticalValue: data.CriticalValue.ValueFloat64(),
		Expr:          eventsExpr,
		Annotations: axonopsClient.MetricAlertAnnotations{
			Description: data.Description.ValueString(),
			Summary: fmt.Sprintf("%s is %s than threshold (current value: {{$value}})",
				data.Name.ValueString(), data.Operator.ValueString()),
		},
		Integrations: axonopsClient.MetricAlertIntegrations{
			Routing: []string{},
		},
	}
}

func (r *logAlertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data logAlertRuleResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(uuid.New().String())

	rule := r.buildRule(&data)

	err := r.client.CreateOrUpdateAlertRule(data.ClusterType.ValueString(), data.ClusterName.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create log alert rule: %s", err))
		return
	}

	tflog.Info(ctx, "Created log alert rule resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *logAlertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data logAlertRuleResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := r.client.GetAlertRules(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read alert rules: %s", err))
		return
	}

	// Find the log alert rule by ID
	var found *axonopsClient.MetricAlertRule
	for _, rule := range rules {
		if rule.ID == data.ID.ValueString() && isLogAlertRule(rule) {
			found = &rule
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	content, level, source, logType := parseEventsExpr(found.Expr)

	data.Name = types.StringValue(found.Alert)
	data.Operator = types.StringValue(found.Operator)
	data.WarningValue = types.Float64Value(found.WarningValue)
	data.CriticalValue = types.Float64Value(found.CriticalValue)
	data.Duration = types.StringValue(found.For)
	// Preserve the prior content value when the API-side value is just the
	// normalised form of what the user wrote — that way the user's natural
	// language config (e.g. "is now DOWN") doesn't drift to the AND-prefixed
	// form (e.g. "+is +now +DOWN") in state and produce noisy plans.
	if normaliseLogContent(data.Content.ValueString()) != content {
		data.Content = types.StringValue(content)
	}
	data.Level = types.StringValue(level)
	data.LogType = types.StringValue(logType)
	data.Source = types.StringValue(source)
	data.Description = types.StringValue(found.Annotations.Description)
	data.Present = types.BoolValue(true)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *logAlertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData logAlertRuleResourceData
	var stateData logAlertRuleResourceData

	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Keep the same ID
	planData.ID = stateData.ID

	rule := r.buildRule(&planData)

	err := r.client.CreateOrUpdateAlertRule(planData.ClusterType.ValueString(), planData.ClusterName.ValueString(), rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update log alert rule: %s", err))
		return
	}

	tflog.Info(ctx, "Updated log alert rule resource")

	diags = resp.State.Set(ctx, &planData)
	resp.Diagnostics.Append(diags...)
}

func (r *logAlertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data logAlertRuleResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAlertRule(data.ClusterType.ValueString(), data.ClusterName.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete log alert rule: %s", err))
		return
	}

	tflog.Info(ctx, "Deleted log alert rule resource")
}

// ImportState imports an existing log alert rule.
// Import ID format: cluster_type/cluster_name/alert_id
func (r *logAlertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: cluster_type/cluster_name/alert_id, got: %s", req.ID),
		)
		return
	}

	clusterType := parts[0]
	clusterName := parts[1]
	alertID := parts[2]

	rules, err := r.client.GetAlertRules(clusterType, clusterName)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Unable to read alert rules: %s", err))
		return
	}

	var found *axonopsClient.MetricAlertRule
	for _, rule := range rules {
		if rule.ID == alertID && isLogAlertRule(rule) {
			found = &rule
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Import Error", fmt.Sprintf("Log alert rule %s not found in cluster %s/%s", alertID, clusterType, clusterName))
		return
	}

	content, level, source, logType := parseEventsExpr(found.Expr)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_type"), clusterType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), found.Alert)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content"), content)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), found.Annotations.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("operator"), found.Operator)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("warning_value"), found.WarningValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("critical_value"), found.CriticalValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("duration"), found.For)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("level"), level)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("log_type"), logType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source"), source)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("present"), true)...)

	tflog.Info(ctx, fmt.Sprintf("Imported log alert rule %s from cluster %s/%s", alertID, clusterType, clusterName))
}
