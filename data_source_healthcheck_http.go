package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*httpHealthcheckDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*httpHealthcheckDataSource)(nil)

type httpHealthcheckDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewHTTPHealthcheckDataSource() datasource.DataSource {
	return &httpHealthcheckDataSource{}
}

func (d *httpHealthcheckDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *httpHealthcheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_healthcheck_http"
}

func (d *httpHealthcheckDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an HTTP healthcheck configuration.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the healthcheck.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the healthcheck.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The URL to check.",
			},
			"method": schema.StringAttribute{
				Computed:    true,
				Description: "The HTTP method.",
			},
			"headers": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "HTTP headers.",
			},
			"body": schema.StringAttribute{
				Computed:    true,
				Description: "The request body.",
			},
			"expected_status": schema.Int64Attribute{
				Computed:    true,
				Description: "The expected HTTP status code.",
			},
			"interval": schema.StringAttribute{
				Computed:    true,
				Description: "The interval between checks.",
			},
			"timeout": schema.StringAttribute{
				Computed:    true,
				Description: "The timeout for the check.",
			},
			"readonly": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the healthcheck is read-only.",
			},
			"supported_agent_types": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of agent types this healthcheck applies to.",
			},
		},
	}
}

type httpHealthcheckDataSourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	URL                 types.String `tfsdk:"url"`
	Method              types.String `tfsdk:"method"`
	Headers             types.Map    `tfsdk:"headers"`
	Body                types.String `tfsdk:"body"`
	ExpectedStatus      types.Int64  `tfsdk:"expected_status"`
	Interval            types.String `tfsdk:"interval"`
	Timeout             types.String `tfsdk:"timeout"`
	Readonly            types.Bool   `tfsdk:"readonly"`
	SupportedAgentTypes types.List   `tfsdk:"supported_agent_types"`
}

func (d *httpHealthcheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data httpHealthcheckDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	healthchecks, err := d.client.GetHealthchecks(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read healthchecks: %s", err))
		return
	}

	var found *axonopsClient.HTTPHealthcheck
	for _, c := range healthchecks.HTTPChecks {
		if c.Name == data.Name.ValueString() {
			found = &c
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("HTTP healthcheck %s not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(found.ID)
	data.URL = types.StringValue(found.URL)
	data.Method = types.StringValue(found.Method)
	data.Body = types.StringValue(found.Body)
	data.ExpectedStatus = types.Int64Value(int64(found.ExpectedStatus))
	data.Interval = types.StringValue(found.Interval)
	data.Timeout = types.StringValue(found.Timeout)
	data.Readonly = types.BoolValue(found.Readonly)

	data.Headers, diags = types.MapValueFrom(ctx, types.StringType, found.Headers)
	resp.Diagnostics.Append(diags...)

	data.SupportedAgentTypes, diags = types.ListValueFrom(ctx, types.StringType, found.SupportedAgentType)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
