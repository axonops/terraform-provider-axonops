package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = (*aclSingleDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*aclSingleDataSource)(nil)

type aclSingleDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

// NewKafkaACLSingleDataSource returns a data source that looks up a single
// Kafka ACL entry by its full identity (cluster_name plus every identity
// field of the axonops_kafka_acl resource). Named distinctly from
// NewKafkaACLDataSource, which backs the existing axonops_kafka_acl_list
// data source (a list of all ACLs for a cluster).
func NewKafkaACLSingleDataSource() datasource.DataSource {
	return &aclSingleDataSource{}
}

func (d *aclSingleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *aclSingleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_acl"
}

func (d *aclSingleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a single Kafka ACL entry matching the given identity fields. Errors if no matching ACL is found.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"resource_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of resource. Valid values: ANY, TOPIC, GROUP, CLUSTER, TRANSACTIONAL_ID, DELEGATION_TOKEN, USER.",
			},
			"resource_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the resource.",
			},
			"resource_pattern_type": schema.StringAttribute{
				Required:    true,
				Description: "The pattern type. Valid values: ANY, MATCH, LITERAL, PREFIXED.",
			},
			"principal": schema.StringAttribute{
				Required:    true,
				Description: "The principal (e.g., User:alice).",
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The host.",
			},
			"operation": schema.StringAttribute{
				Required:    true,
				Description: "The operation. Valid values: ANY, ALL, READ, WRITE, CREATE, DELETE, ALTER, DESCRIBE, CLUSTER_ACTION, DESCRIBE_CONFIGS, ALTER_CONFIGS, IDEMPOTENT_WRITE, CREATE_TOKENS, DESCRIBE_TOKENS.",
			},
			"permission_type": schema.StringAttribute{
				Required:    true,
				Description: "The permission type. Valid values: ANY, DENY, ALLOW.",
			},
		},
	}
}

func (d *aclSingleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data aclResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aclResponse, err := d.client.GetACLs(ctx, data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACLs: %s", err))
		return
	}

	if !findACL(data, aclResponse) {
		resp.Diagnostics.AddError(
			"ACL Not Found",
			fmt.Sprintf("No ACL matching the given identity fields was found in cluster %s", data.ClusterName.ValueString()),
		)
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
