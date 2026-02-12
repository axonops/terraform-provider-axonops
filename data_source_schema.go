package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*schemaDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*schemaDataSource)(nil)

type schemaDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSchemaDataSource() datasource.DataSource {
	return &schemaDataSource{}
}

func (d *schemaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *schemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schema"
}

func (d *schemaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Schema Registry schema subject.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"subject": schema.StringAttribute{
				Required:    true,
				Description: "The subject name.",
			},
			"schema": schema.StringAttribute{
				Computed:    true,
				Description: "The schema definition.",
			},
			"schema_type": schema.StringAttribute{
				Computed:    true,
				Description: "The schema type (AVRO, PROTOBUF, JSON).",
			},
			"schema_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The unique ID assigned to the schema.",
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "The version number of the schema.",
			},
		},
	}
}

type schemaDataSourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	Subject     types.String `tfsdk:"subject"`
	Schema      types.String `tfsdk:"schema"`
	SchemaType  types.String `tfsdk:"schema_type"`
	SchemaId    types.Int64  `tfsdk:"schema_id"`
	Version     types.Int64  `tfsdk:"version"`
}

func (d *schemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data schemaDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.GetSchema(data.ClusterName.ValueString(), data.Subject.ValueString(), "latest")
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read schema: %s", err))
		return
	}

	if result == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Schema subject %s not found", data.Subject.ValueString()))
		return
	}

	data.Schema = types.StringValue(result.Schema)
	data.SchemaType = types.StringValue(result.Type)
	data.SchemaId = types.Int64Value(int64(result.Id))
	data.Version = types.Int64Value(int64(result.Version))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
