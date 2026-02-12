package main

import (
	"context"

	axonopsClient "axonops-tf/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = (*axonopsProvider)(nil)

// var _ provider.ProviderWithMetadata = (*axonopsProvider)(nil)

type axonopsProvider struct{}

type axonopsProviderModel struct {
	ApiKey          types.String `tfsdk:"api_key"`
	AxonopsHost     types.String `tfsdk:"axonops_host"`
	AxonopsProtocol types.String `tfsdk:"axonops_protocol"`
	OrgId           types.String `tfsdk:"org_id"`
	TokenType       types.String `tfsdk:"token_type"`
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &axonopsProvider{}
	}
}

func (p *axonopsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config axonopsProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var protocol = "https"
	var axonopsHost = ""
	var apiKey = ""
	var tokenType = "Bearer"

	if !config.AxonopsProtocol.IsNull() {
		protocol = config.AxonopsProtocol.ValueString()
	}

	if !config.AxonopsHost.IsNull() {
		axonopsHost = config.AxonopsHost.ValueString()
	}

	// Default axonops_host uses org_id: dash.axonops.cloud/<org_id>
	if axonopsHost == "" {
		axonopsHost = "dash.axonops.cloud/" + config.OrgId.ValueString()
	}

	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}

	if !config.TokenType.IsNull() {
		tokenType = config.TokenType.ValueString()
		if tokenType != "AxonApi" && tokenType != "Bearer" {
			resp.Diagnostics.AddAttributeError(
				path.Root("token_type"),
				"Invalid Token Type",
				"token_type must be either 'AxonApi' or 'Bearer'",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := axonopsClient.CreateHTTPClient(protocol, axonopsHost, apiKey, config.OrgId.ValueString(), tokenType)

	if client == nil {
		tflog.Error(ctx, "Client not initialised")
		resp.Diagnostics.AddAttributeError(
			path.Root("http_client"),
			"Error creating connection to AxonOps",
			"Failed to initialise HTTP client for AxonOps API",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.ResourceData = client

}

func (p *axonopsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "axonops"
}

func (p *axonopsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewKafkaTopicDataSource,
		NewKafkaACLDataSource,
		NewKafkaConnectConnectorDataSource,
		NewSchemaDataSource,
		NewLogCollectorDataSource,
		NewTCPHealthcheckDataSource,
		NewHTTPHealthcheckDataSource,
		NewShellHealthcheckDataSource,
		NewCassandraAdaptiveRepairDataSource,
		NewCassandraBackupDataSource,
		NewMetricAlertRuleDataSource,
	}
}

func (p *axonopsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKafkaTopicResource,
		NewKafkaACLResource,
		NewKafkaConnectConnectorResource,
		NewSchemaResource,
		NewLogCollectorResource,
		NewTCPHealthcheckResource,
		NewHTTPHealthcheckResource,
		NewShellHealthcheckResource,
		NewCassandraAdaptiveRepairResource,
		NewCassandraBackupResource,
		NewMetricAlertRuleResource,
		NewAlertRouteResource,
	}
}

func (p *axonopsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional: true,
			},
			"axonops_host": schema.StringAttribute{
				Optional:    true,
				Description: "AxonOps server hostname. Default: dash.axonops.cloud/<org_id>",
			},
			"axonops_protocol": schema.StringAttribute{
				Optional: true,
			},
			"org_id": schema.StringAttribute{
				Required: true,
			},
			"token_type": schema.StringAttribute{
				Optional:    true,
				Description: "Token type for Authorization header. Valid values: 'Bearer' (default) or 'AxonApi'",
			},
		},
	}
}
