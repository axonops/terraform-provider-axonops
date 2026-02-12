package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*tcpHealthcheckDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*tcpHealthcheckDataSource)(nil)

type tcpHealthcheckDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewTCPHealthcheckDataSource() datasource.DataSource {
	return &tcpHealthcheckDataSource{}
}

func (d *tcpHealthcheckDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tcpHealthcheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_healthcheck_tcp"
}

func (d *tcpHealthcheckDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a TCP healthcheck configuration.",
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
			"tcp": schema.StringAttribute{
				Computed:    true,
				Description: "The TCP address to check.",
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

type tcpHealthcheckDataSourceData struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	TCP                 types.String `tfsdk:"tcp"`
	Interval            types.String `tfsdk:"interval"`
	Timeout             types.String `tfsdk:"timeout"`
	Readonly            types.Bool   `tfsdk:"readonly"`
	SupportedAgentTypes types.List   `tfsdk:"supported_agent_types"`
}

func (d *tcpHealthcheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tcpHealthcheckDataSourceData

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

	var found *axonopsClient.TCPHealthcheck
	for _, c := range healthchecks.TCPChecks {
		if c.Name == data.Name.ValueString() {
			found = &c
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("TCP healthcheck %s not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(found.ID)
	data.TCP = types.StringValue(found.TCP)
	data.Interval = types.StringValue(found.Interval)
	data.Timeout = types.StringValue(found.Timeout)
	data.Readonly = types.BoolValue(found.Readonly)

	data.SupportedAgentTypes, diags = types.ListValueFrom(ctx, types.StringType, found.SupportedAgentType)
	resp.Diagnostics.Append(diags...)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
