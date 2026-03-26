package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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
}

// samlCache stores per-org SAML detection results to avoid repeated probes
// within the same provider process (e.g. across plan and apply).
var (
	samlCache   = map[string]bool{}
	samlCacheMu sync.RWMutex
)

// detectSAML probes {protocol}://{host}/dashboard/ to determine whether the
// host is a SAML-enabled AxonOps deployment. It returns true if the server
// responds with any HTTP status (including 401/403/302), and false if the
// connection fails or returns 404. Results are cached by host so the probe
// is only made once per host per process.
func detectSAML(protocol, host string, tlsSkipVerify bool) bool {
	cacheKey := protocol + ":" + host

	samlCacheMu.RLock()
	if cached, ok := samlCache[cacheKey]; ok {
		samlCacheMu.RUnlock()
		if os.Getenv("AXONOPS_DEBUG") != "" {
			fmt.Printf("[AXONOPS DEBUG] SAML detection for host %q: cached=%v\n", host, cached)
		}
		return cached
	}
	samlCacheMu.RUnlock()

	probeURL := fmt.Sprintf("%s://%s/dashboard/", protocol, host)
	if os.Getenv("AXONOPS_DEBUG") != "" {
		fmt.Printf("[AXONOPS DEBUG] SAML detection: probing %s\n", probeURL)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsSkipVerify},
	}
	c := &http.Client{
		Timeout:   5 * time.Second,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := c.Get(probeURL)
	// SAML /dashboard/ returns JSON (IDP redirect payload).
	// On-prem servers serve the SPA as HTML at that path.
	// So we only consider it SAML if the response is JSON.
	isSAML := err == nil && resp != nil &&
		resp.StatusCode != http.StatusNotFound &&
		strings.Contains(resp.Header.Get("Content-Type"), "application/json")
	if resp != nil {
		resp.Body.Close() // #nosec G104 -- error from Body.Close is intentionally ignored
	}

	if os.Getenv("AXONOPS_DEBUG") != "" {
		statusCode := 0
		contentType := ""
		if resp != nil {
			statusCode = resp.StatusCode
			contentType = resp.Header.Get("Content-Type")
		}
		fmt.Printf("[AXONOPS DEBUG] SAML detection for host %q: isSAML=%v (status=%d, content-type=%q, err=%v)\n", host, isSAML, statusCode, contentType, err)
	}

	samlCacheMu.Lock()
	samlCache[cacheKey] = isSAML
	samlCacheMu.Unlock()

	return isSAML
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

	if !config.AxonopsProtocol.IsNull() {
		protocol = config.AxonopsProtocol.ValueString()
	}

	if !config.AxonopsHost.IsNull() {
		axonopsHost = config.AxonopsHost.ValueString()
	}

	if !config.TlsSkipVerify.IsNull() {
		tlsSkipVerify = config.TlsSkipVerify.ValueBool()
	}

	// Construct axonops_host based on configuration. SAML is auto-detected
	// in both cases by probing {host}/dashboard/.
	//
	// No custom host:
	//   SAML org:     {org_id}.axonops.cloud/dashboard
	//   Non-SAML org: dash.axonops.cloud/{org_id}
	// Custom host:
	//   SAML:         {custom_host}/dashboard
	//   Non-SAML:     {custom_host}
	if axonopsHost == "" {
		orgId := config.OrgId.ValueString()
		samlHost := orgId + ".axonops.cloud"
		if detectSAML(protocol, samlHost, tlsSkipVerify) {
			axonopsHost = samlHost + "/dashboard"
		} else {
			axonopsHost = "dash.axonops.cloud/" + orgId
		}
	} else {
		if detectSAML(protocol, axonopsHost, tlsSkipVerify) {
			axonopsHost = axonopsHost + "/dashboard"
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
		NewCassandraScheduledRepairDataSource,
		NewSilenceDataSource,
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
		NewCassandraScheduledRepairResource,
		NewSilenceResource,
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
				Description: "AxonOps server hostname (without protocol). For SaaS, leave empty to auto-detect the correct URL. For on-premise deployments, specify your server hostname. Can also be set via AXONOPS_HOST environment variable.",
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
		},
	}
}
