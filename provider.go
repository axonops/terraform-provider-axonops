package main

import (
	"context"
	"os"

	axonopsClient "terraform-provider-axonops/client"

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
	TlsSkipVerify   types.Bool   `tfsdk:"tls_skip_verify"`
	OrgId           types.String `tfsdk:"org_id"`
	TokenType       types.String `tfsdk:"token_type"`
	UseSaml         types.Bool   `tfsdk:"use_saml"`
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &axonopsProvider{}
	}
}

func getEnvOrDefault(variableName string, defaultValue string) string {
	if value, exists := os.LookupEnv(variableName); exists {
		return value
	}
	return defaultValue
}

func (p *axonopsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config axonopsProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var protocol = getEnvOrDefault("AXONOPS_PROTOCOL", "https")
	var axonopsHost = getEnvOrDefault("AXONOPS_HOST", "")
	var apiKey = getEnvOrDefault("AXONOPS_API_KEY", "")
	var tokenType = getEnvOrDefault("AXONOPS_TOKEN_TYPE", "Bearer")
	var tlsSkipVerify = getEnvOrDefault("AXONOPS_TLS_SKIP_VERIFY", "false") == "true"
	var useSaml = getEnvOrDefault("AXONOPS_USE_SAML", "false") == "true"

	if !config.AxonopsProtocol.IsNull() {
		protocol = config.AxonopsProtocol.ValueString()
	}

	if !config.AxonopsHost.IsNull() {
		axonopsHost = config.AxonopsHost.ValueString()
	}

	if !config.TlsSkipVerify.IsNull() {
		tlsSkipVerify = config.TlsSkipVerify.ValueBool()
	}

	if !config.UseSaml.IsNull() {
		useSaml = config.UseSaml.ValueBool()
	}

	// Construct axonops_host based on SAML configuration
	// SAML enabled: {org_id}.axonops.cloud/dashboard
	// SAML disabled: dash.axonops.cloud/{org_id}
	// Custom host with SAML: {custom_host}/dashboard
	// Custom host without SAML: {custom_host}/{org_id}
	if axonopsHost == "" {
		if useSaml {
			axonopsHost = config.OrgId.ValueString() + ".axonops.cloud/dashboard"
		} else {
			axonopsHost = "dash.axonops.cloud/" + config.OrgId.ValueString()
		}
	} else {
		// Custom host provided
		if useSaml {
			axonopsHost = axonopsHost + "/dashboard"
		} else {
			axonopsHost = axonopsHost + "/" + config.OrgId.ValueString()
		}
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

	client := axonopsClient.CreateHTTPClient(protocol, axonopsHost, apiKey, config.OrgId.ValueString(), tokenType, tlsSkipVerify)

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
		NewLogAlertRuleDataSource,
		NewSlackIntegrationDataSource,
		NewTeamsIntegrationDataSource,
		NewPagerDutyIntegrationDataSource,
		NewOpsGenieIntegrationDataSource,
		NewServiceNowIntegrationDataSource,
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
		NewLogAlertRuleResource,
		NewSlackIntegrationResource,
		NewTeamsIntegrationResource,
		NewPagerDutyIntegrationResource,
		NewOpsGenieIntegrationResource,
		NewServiceNowIntegrationResource,
	}
}

func (p *axonopsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Description: "API key for authentication. Can also be set via AXONOPS_API_KEY environment variable.",
			},
			"axonops_host": schema.StringAttribute{
				Optional:    true,
				Description: "AxonOps server hostname (without protocol). For SaaS, leave empty to use the default. For on-premise deployments, specify your server hostname. Can also be set via AXONOPS_HOST environment variable.",
			},
			"axonops_protocol": schema.StringAttribute{
				Optional:    true,
				Description: "Protocol to use for API requests. Valid values: 'https' (default) or 'http'. Can also be set via AXONOPS_PROTOCOL environment variable.",
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization ID for your AxonOps account.",
			},
			"tls_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification. Use with caution, only for self-signed certificates. Default: false. Can also be set via AXONOPS_TLS_SKIP_VERIFY environment variable.",
			},
			"token_type": schema.StringAttribute{
				Optional:    true,
				Description: "Token type for Authorization header. Valid values: 'Bearer' (default for SaaS) or 'AxonApi' (for on-premise). Can also be set via AXONOPS_TOKEN_TYPE environment variable.",
			},
			"use_saml": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable SAML authentication mode. When enabled, uses tenant-specific URL routing ({org_id}.axonops.cloud/dashboard). Default: false. Can also be set via AXONOPS_USE_SAML environment variable.",
			},
		},
	}
}
