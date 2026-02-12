package main

import (
	"context"
	"fmt"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*aclDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*aclDataSource)(nil)

type aclDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaACLDataSource() datasource.DataSource {
	return &aclDataSource{}
}

func (d *aclDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *aclDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_acl_list"
}

func (d *aclDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Kafka ACLs for a cluster.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"acls": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of ACL entries.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"resource_type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of resource.",
						},
						"resource_name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the resource.",
						},
						"resource_pattern_type": schema.StringAttribute{
							Computed:    true,
							Description: "The pattern type.",
						},
						"principal": schema.StringAttribute{
							Computed:    true,
							Description: "The principal.",
						},
						"host": schema.StringAttribute{
							Computed:    true,
							Description: "The host.",
						},
						"operation": schema.StringAttribute{
							Computed:    true,
							Description: "The operation.",
						},
						"permission_type": schema.StringAttribute{
							Computed:    true,
							Description: "The permission type.",
						},
					},
				},
			},
		},
	}
}

type aclDataSourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	ACLs        []aclEntry  `tfsdk:"acls"`
}

type aclEntry struct {
	ResourceType        types.String `tfsdk:"resource_type"`
	ResourceName        types.String `tfsdk:"resource_name"`
	ResourcePatternType types.String `tfsdk:"resource_pattern_type"`
	Principal           types.String `tfsdk:"principal"`
	Host                types.String `tfsdk:"host"`
	Operation           types.String `tfsdk:"operation"`
	PermissionType      types.String `tfsdk:"permission_type"`
}

func (d *aclDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data aclDataSourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aclResponse, err := d.client.GetACLs(data.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACLs: %s", err))
		return
	}

	var entries []aclEntry
	for _, res := range aclResponse.ACLResources {
		for _, acl := range res.ACLs {
			entries = append(entries, aclEntry{
				ResourceType:        types.StringValue(res.ResourceType),
				ResourceName:        types.StringValue(res.ResourceName),
				ResourcePatternType: types.StringValue(res.ResourcePatternType),
				Principal:           types.StringValue(acl.Principal),
				Host:                types.StringValue(acl.Host),
				Operation:           types.StringValue(acl.Operation),
				PermissionType:      types.StringValue(acl.PermissionType),
			})
		}
	}

	if entries == nil {
		entries = []aclEntry{}
	}
	data.ACLs = entries

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
