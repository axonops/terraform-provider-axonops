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
// host is a SAML-enabled AxonOps deployment. SAML deployments answer that path
// with JSON (the IDP redirect payload); on-prem servers serve the SPA as HTML.
// Conclusive results are cached by host so the probe is only made once per
// host per process. Network errors are not cached, so a transient failure
// does not pin the wrong URL layout for the rest of the run.
func detectSAML(ctx context.Context, protocol, host string, tlsSkipVerify bool) bool {
	cacheKey := protocol + ":" + host

	samlCacheMu.RLock()
	if cached, ok := samlCache[cacheKey]; ok {
		samlCacheMu.RUnlock()
		tflog.Debug(ctx, "SAML detection (cached)", map[string]interface{}{"host": host, "saml": cached})
		return cached
	}
	samlCacheMu.RUnlock()

	probeURL := fmt.Sprintf("%s://%s/dashboard/", protocol, host)
	tflog.Debug(ctx, "SAML detection: probing", map[string]interface{}{"url": probeURL})

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: tlsSkipVerify}, // #nosec G402 -- opt-in via tls_skip_verify; a warning diagnostic is emitted
	}
	c := &http.Client{
		Timeout:   5 * time.Second,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		tflog.Warn(ctx, "SAML detection: invalid probe URL, assuming non-SAML", map[string]interface{}{"url": probeURL, "error": err.Error()})
		return false
	}
	resp, err := c.Do(req)
	if err != nil {
		tflog.Warn(ctx, "SAML detection probe failed, assuming non-SAML (not cached)", map[string]interface{}{"url": probeURL, "error": err.Error()})
		return false
	}
	_ = resp.Body.Close()

	isSAML := resp.StatusCode != http.StatusNotFound &&
		strings.Contains(resp.Header.Get("Content-Type"), "application/json")
	tflog.Debug(ctx, "SAML detection result", map[string]interface{}{
		"host": host, "saml": isSAML, "status": resp.StatusCode, "content_type": resp.Header.Get("Content-Type"),
	})

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

	for attr, unknown := range map[string]bool{
		"api_key":          config.ApiKey.IsUnknown(),
		"axonops_host":     config.AxonopsHost.IsUnknown(),
		"axonops_protocol": config.AxonopsProtocol.IsUnknown(),
		"org_id":           config.OrgId.IsUnknown(),
		"tls_skip_verify":  config.TlsSkipVerify.IsUnknown(),
		"token_type":       config.TokenType.IsUnknown(),
	} {
		if unknown {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Unknown Provider Configuration Value",
				fmt.Sprintf("The provider cannot be configured because %q is not known until apply. Set it statically or via environment variable.", attr))
		}
	}
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

	if protocol != "https" && protocol != "http" {
		resp.Diagnostics.AddAttributeError(path.Root("axonops_protocol"), "Invalid Protocol",
			fmt.Sprintf("axonops_protocol must be 'https' or 'http', got %q", protocol))
		return
	}

	if tlsSkipVerify {
		resp.Diagnostics.AddWarning("TLS Certificate Verification Disabled",
			"tls_skip_verify is enabled. The provider will not verify the AxonOps server certificate, "+
				"which exposes the API key to man-in-the-middle attacks. Use only with self-signed certificates in trusted networks.")
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
		if detectSAML(ctx, protocol, samlHost, tlsSkipVerify) {
			axonopsHost = samlHost + "/dashboard"
		} else {
			axonopsHost = "dash.axonops.cloud/" + orgId
		}
	} else {
		if detectSAML(ctx, protocol, axonopsHost, tlsSkipVerify) {
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
		NewAlertRouteDataSource,
		NewKafkaACLSingleDataSource,
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
				Sensitive:   true,
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
