package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = (*silenceDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*silenceDataSource)(nil)

type silenceDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSilenceDataSource() datasource.DataSource {
	return &silenceDataSource{}
}

func (d *silenceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *silenceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_silence"
}

func (d *silenceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a silence window configuration.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster.",
			},
			"cluster_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of cluster (e.g., cassandra, kafka).",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the silence window.",
			},
			"active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the silence is active.",
			},
			"is_recurring": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the silence is recurring based on the cron expression.",
			},
			"cron_expr": schema.StringAttribute{
				Computed:    true,
				Description: "Cron expression for recurring silences.",
			},
			"duration": schema.StringAttribute{
				Computed:    true,
				Description: "Duration of the silence.",
			},
			"datacenters": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of datacenters or nodes the silence applies to.",
			},
		},
	}
}

type silenceDataSourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	ClusterType types.String `tfsdk:"cluster_type"`
	ID          types.String `tfsdk:"id"`
	Active      types.Bool   `tfsdk:"active"`
	IsRecurring types.Bool   `tfsdk:"is_recurring"`
	CronExpr    types.String `tfsdk:"cron_expr"`
	Duration    types.String `tfsdk:"duration"`
	Datacenters types.List   `tfsdk:"datacenters"`
}

func (d *silenceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data silenceDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	silences, err := d.client.GetSilenceWindows(data.ClusterType.ValueString(), data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read silences: %s", err))
		return
	}

	found := axonopsClient.FindSilenceWindowByID(silences, data.ID.ValueString())
	if found == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Silence %s not found", data.ID.ValueString()))
		return
	}

	data.Active = types.BoolValue(found.Active)
	data.IsRecurring = types.BoolValue(found.IsRecurring)
	data.CronExpr = types.StringValue(found.CronExpr)
	data.Duration = types.StringValue(found.Duration)

	dcs := found.DCs
	if dcs == nil {
		dcs = []string{}
	}
	data.Datacenters, diags = types.ListValueFrom(ctx, types.StringType, dcs)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, fmt.Sprintf("Read silence %s", data.ID.ValueString()))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
