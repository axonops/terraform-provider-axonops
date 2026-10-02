package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*integrationsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*integrationsDataSource)(nil)

type integrationsDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewIntegrationsDataSource() datasource.DataSource {
	return &integrationsDataSource{}
}

func (d *integrationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *integrationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integrations"
}

func (d *integrationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the alert integrations (Slack, Microsoft Teams, PagerDuty, OpsGenie, ServiceNow, ...) configured for a cluster. " +
			"Secret parameters (webhook URLs, keys, passwords) are never exposed.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The cluster type (cassandra, kafka, or dse).",
				Validators:  []validator.String{clusterTypeValidator()},
			},
			"type": schema.StringAttribute{
				Optional: true,
				Description: "Only return integrations of this type, case-insensitive. " +
					"Examples: `slack`, `microsoft_teams`, `pagerduty`, `opsgenie`, `servicenow`.",
			},
			"integrations": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Matching integrations, sorted by type then name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The integration ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The integration name.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The integration type as reported by the API.",
						},
					},
				},
			},
		},
	}
}

type integrationsDataSourceData struct {
	ClusterName  types.String            `tfsdk:"cluster_name"`
	ClusterType  types.String            `tfsdk:"cluster_type"`
	Type         types.String            `tfsdk:"type"`
	Integrations []integrationsListEntry `tfsdk:"integrations"`
}

type integrationsListEntry struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (d *integrationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data integrationsDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetIntegrations(ctx, data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list integrations: %s", err))
		return
	}

	defs := append([]axonopsClient.IntegrationDefinition(nil), result.Definitions...)
	sort.Slice(defs, func(i, j int) bool {
		if defs[i].Type != defs[j].Type {
			return defs[i].Type < defs[j].Type
		}
		if defs[i].Params["name"] != defs[j].Params["name"] {
			return defs[i].Params["name"] < defs[j].Params["name"]
		}
		return defs[i].ID < defs[j].ID
	})

	entries := []integrationsListEntry{}
	for _, def := range defs {
		if !data.Type.IsNull() && !strings.EqualFold(data.Type.ValueString(), def.Type) {
			continue
		}
		entries = append(entries, integrationsListEntry{
			ID:   types.StringValue(def.ID),
			Name: types.StringValue(def.Params["name"]),
			Type: types.StringValue(def.Type),
		})
	}
	data.Integrations = entries

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
