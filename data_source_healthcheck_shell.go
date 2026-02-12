package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*shellHealthcheckDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*shellHealthcheckDataSource)(nil)

type shellHealthcheckDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewShellHealthcheckDataSource() datasource.DataSource {
	return &shellHealthcheckDataSource{}
}

func (d *shellHealthcheckDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *shellHealthcheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_healthcheck_shell"
}

func (d *shellHealthcheckDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a shell healthcheck configuration.",
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
			"script": schema.StringAttribute{
				Computed:    true,
				Description: "The script or command to execute.",
			},
			"shell": schema.StringAttribute{
				Computed:    true,
				Description: "The shell to use for executing the script.",
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
		},
	}
}

type shellHealthcheckDataSourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	Script      types.String `tfsdk:"script"`
	Shell       types.String `tfsdk:"shell"`
	Interval    types.String `tfsdk:"interval"`
	Timeout     types.String `tfsdk:"timeout"`
	Readonly    types.Bool   `tfsdk:"readonly"`
}

func (d *shellHealthcheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data shellHealthcheckDataSourceData

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

	var found *axonopsClient.ShellHealthcheck
	for _, c := range healthchecks.ShellChecks {
		if c.Name == data.Name.ValueString() {
			found = &c
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Shell healthcheck %s not found", data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(found.ID)
	data.Script = types.StringValue(found.Script)
	data.Shell = types.StringValue(found.Shell)
	data.Interval = types.StringValue(found.Interval)
	data.Timeout = types.StringValue(found.Timeout)
	data.Readonly = types.BoolValue(found.Readonly)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
