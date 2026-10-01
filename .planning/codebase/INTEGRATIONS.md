# External Integrations

**Analysis Date:** 2026-05-14

## APIs & External Services

**AxonOps Platform (the only first-class external API):**
- Base URL pattern: `{protocol}://{axonopsHost}/api/v1/{orgId}/...` (built in `client/http_client.go`, constant `axonops_api_version = "api/v1"`)
- SaaS hosts: `dash.axonops.cloud/{orgId}` (token auth) or `{orgId}.axonops.cloud/dashboard` (SAML)
- On-prem: user-supplied host
- Auto-detection of SAML in `provider.go` `detectSAML()` — probes `{protocol}://{host}/dashboard/` and treats a JSON response as SAML-enabled (cached per host via `samlCache`)
- Auth: `Authorization: {tokenType} {apiKey}` where `tokenType` ∈ {`Bearer`, `AxonApi`} — set in every request builder in `client/http_client.go`

**Alerting integrations (managed BY this provider, not consumed by it):**
- Slack — `resource_slack_integration.go` / `data_source_slack_integration.go`
- Microsoft Teams — `resource_teams_integration.go`
- PagerDuty — `resource_pagerduty_integration.go`
- OpsGenie — `resource_opsgenie_integration.go`
- ServiceNow — `resource_servicenow_integration.go`
- All routed via `AxonopsHttpClient.CreateOrUpdateIntegration` / `AddIntegrationRoute` / `RemoveIntegrationRoute` / `SetIntegrationOverride` in `client/http_client.go`

## Data Storage

**Databases:**
- None — provider is stateless; Terraform state file is the only persistence on the consumer side
- Targets observed by AxonOps: Apache Cassandra and Apache Kafka (resources for topics, ACLs, connectors, schemas, repairs, backups)

**File Storage:**
- Local filesystem only — Terraform state and provider binary cache

**Caching:**
- In-memory `samlCache` in `provider.go` (per-process map of host → bool, guarded by `samlCacheMu sync.RWMutex`). Avoids re-probing SAML between plan and apply.

## Authentication & Identity

**Auth Provider:**
- AxonOps API key (Bearer or AxonApi token type)
- SAML auto-detected on the AxonOps server side; the provider only switches the URL path (`/dashboard` suffix) when SAML is detected
- No OAuth flow, no session handling; every request carries the static token

## Monitoring & Observability

**Error Tracking:**
- None embedded
- `tflog.Error` / `tflog.Debug` via `github.com/hashicorp/terraform-plugin-log` — surfaces in Terraform's `TF_LOG` output

**Logs:**
- `tflog` for structured Terraform logs
- Manual `fmt.Printf("[AXONOPS DEBUG] ...")` gated by `AXONOPS_DEBUG` env var in `client/http_client.go` (`debugRequest`, `debugResponse`) and `provider.go` (`detectSAML`)
- Authorization header is masked in debug output (`http_client.go:30-35`)

## CI/CD & Deployment

**Hosting:**
- Terraform Registry: `registry.terraform.io/axonops/axonops` (declared in `main.go`)
- GitHub Releases — binaries + signed checksums

**CI Pipeline (`.github/workflows/`):**
- `test.yml` — unit tests on Go 1.23 and 1.24, plus acceptance tests with `TF_ACC=1`
- `build.yml` — build verification
- `code-quality.yml` — lint
- `dependencies.yml` — Dependabot / dependency review
- `docs.yml` — verifies `tfplugindocs` output is up to date
- `release.yml` — triggered by `v*` tags; runs `goreleaser release --clean` with GPG signing

## Environment Configuration

**Required env vars (or equivalent provider attributes):**
- `AXONOPS_API_KEY` (or `api_key` attribute)
- `org_id` provider attribute (Required, no env fallback)
- `AXONOPS_HOST` for on-prem; optional for SaaS

**Optional env vars:**
- `AXONOPS_PROTOCOL`, `AXONOPS_TOKEN_TYPE`, `AXONOPS_TLS_SKIP_VERIFY`, `AXONOPS_DEBUG`

**Secrets location:**
- GitHub Actions secrets: `GPG_PRIVATE_KEY`, `PASSPHRASE`, `GITHUB_TOKEN` (release workflow)
- No `.env` file in repo

## Webhooks & Callbacks

**Incoming:**
- None — provider is a CLI plugin invoked by Terraform; no listening sockets

**Outgoing:**
- None directly — alerting webhooks (Slack, Teams, PagerDuty, OpsGenie, ServiceNow) are configured *into* AxonOps via the API; AxonOps fires them, not this provider

---

*Integration audit: 2026-05-14*
