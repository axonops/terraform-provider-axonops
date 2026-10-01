<!-- refreshed: 2026-05-14 -->
# Architecture

**Analysis Date:** 2026-05-14

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                  Terraform / OpenTofu CLI                    │
│              (invokes provider over gRPC plugin)             │
└─────────────────────────┬───────────────────────────────────┘
                          │ providerserver.Serve
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                 Provider entry point                         │
│  `main.go`  →  `provider.go` (axonopsProvider)               │
│   - Schema(), Configure(), Resources(), DataSources()        │
│   - Auto-detects SAML, builds AxonopsHttpClient              │
└────────┬────────────────────────────────────┬───────────────┘
         │                                    │
         ▼                                    ▼
┌──────────────────────┐          ┌──────────────────────────┐
│  Resource layer      │          │  Data source layer       │
│  `resource_*.go`     │          │  `data_source_*.go`      │
│  (20 resources)      │          │  (19 data sources)       │
│  CRUD + plan modifier│          │  Read only               │
└──────────┬───────────┘          └────────────┬─────────────┘
           │                                   │
           └─────────────┬─────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              AxonOps API client package                      │
│  `client/http_client.go`  (~2100 lines, 41+ methods)         │
│  `client/delete_retry.go` (502/503/504 + 404 idempotent)     │
│  - Owns http.Client, TLS config, auth header                 │
│  - 59 typed request/response structs                         │
└─────────────────────────┬───────────────────────────────────┘
                          │ HTTPS
                          ▼
┌─────────────────────────────────────────────────────────────┐
│   AxonOps SaaS or on-prem  (api/v1/{orgId}/...)              │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Provider entry | gRPC serve, debug flag | `main.go` |
| Provider | Schema, Configure, Resources(), DataSources(), SAML detect | `provider.go` |
| HTTP client | Auth, URL building, JSON marshalling, debug logging, payload structs | `client/http_client.go` |
| DELETE retry | Idempotent DELETE with exponential backoff on 5xx | `client/delete_retry.go` |
| Alert ID helpers | Deterministic UUID v5 + name-based reconciliation for alert rules | `alert_rule_id.go` |
| Resource (`resource_*.go`) | Schema, CRUD, plan modifiers, import state for one Terraform resource | e.g. `resource_metric_alert_rule.go` |
| Data source (`data_source_*.go`) | Schema + Read for one Terraform data source | e.g. `data_source_metric_alert_rule.go` |
| Docs templates | Source for `tfplugindocs` generation | `templates/` |

## Pattern Overview

**Overall:** HashiCorp Terraform Plugin Framework provider with a flat package layout — the `main` package contains the provider plus every resource and data source side-by-side; a single `client` subpackage encapsulates all AxonOps API calls.

**Key Characteristics:**
- One file per resource and one file per data source (filename mirrors the Terraform type, e.g. `axonops_kafka_topic` ↔ `resource_kafka_topic.go`)
- Strict dependency direction: resources/data sources import `terraform-provider-axonops/client`; client imports nothing from `main`
- All HTTP calls funnel through methods on `*AxonopsHttpClient` — no ad-hoc `http.Get` outside the `client` package (sole exception: SAML probe in `provider.go`)
- Reconciliation by name not by ID for alert rules, because the AxonOps API ignores client-supplied IDs on POST (see `alert_rule_id.go` `findAlertRuleByName`)
- Deterministic UUID v5 IDs derived from `org/clusterType/clusterName/alertName/kind` to make Create idempotent against state loss (`alert_rule_id.go` `deterministicAlertRuleID`)

## Layers

**Provider configuration layer:**
- Purpose: parse provider block + env vars, decide URL shape, build the API client
- Location: `provider.go`, `main.go`
- Contains: `axonopsProvider`, `axonopsProviderModel`, `detectSAML`, `samlCache`
- Depends on: `client` package, `terraform-plugin-framework`
- Used by: Terraform CLI via `providerserver.Serve`

**Resource / data source layer:**
- Purpose: translate Terraform schema ↔ API payloads; implement CRUD lifecycle
- Location: `resource_*.go`, `data_source_*.go` (root package `main`)
- Contains: per-resource `*Resource` / `*DataSource` structs, schema definitions, `Create/Read/Update/Delete/ImportState`
- Depends on: `client` package
- Used by: registered via `provider.go` `Resources()` / `DataSources()` slices

**API client layer:**
- Purpose: HTTP transport, authentication, JSON (de)serialisation, retry policy
- Location: `client/`
- Contains: `AxonopsHttpClient`, payload structs (KafkaTopic, MetricAlertRule, IntegrationsResponse, etc.), `doDeleteWithRetry`
- Depends on: standard library only
- Used by: provider + every resource and data source

## Data Flow

### Primary Request Path (e.g. `terraform apply` creating a Kafka topic)

1. Terraform CLI invokes provider over gRPC (`main.go:27` `providerserver.Serve`)
2. Provider `Configure` resolves env vars, runs `detectSAML` if needed, builds `AxonopsHttpClient` (`provider.go:121-206`)
3. Client stored in `resp.ResourceData`; each resource's `Configure` extracts it (`resource_*.go` `Configure`)
4. `kafkaTopicResource.Create` reads the plan model, calls `r.client.CreateTopic(...)` (`resource_kafka_topic.go`)
5. `AxonopsHttpClient.CreateTopic` builds URL `{protocol}://{host}/api/v1/{orgId}/kafka/{cluster}/topics`, marshals JSON, sends POST with `Authorization` header (`client/http_client.go:124-169`)
6. Response parsed; resource writes computed attributes back to state via `resp.State.Set(...)`

### DELETE flow (idempotent retry)

1. Resource `Delete` calls e.g. `client.DeleteAlertRule(...)`
2. Client builds DELETE request, calls `doDeleteWithRetry(req, body)` (`client/delete_retry.go:51-92`)
3. On transport error or 502/503/504: exponential backoff (1s, 2s, 4s) up to 3 retries, honouring request context cancellation
4. 404 treated as success (idempotent — already gone)
5. 4xx other than 404 returned immediately (definitive rejection)

### Alert rule reconciliation

1. `Create` POSTs the rule; AxonOps returns its own UUID, but the provider ignores it
2. `Read` lists all rules via `GetAlertRules`, then `findAlertRuleByName` matches by `Alert` name + kind (`alert_rule_id.go:24-31`)
3. State `id` is the deterministic UUID v5 from `deterministicAlertRuleID(...)`, NOT the API-supplied ID — so Create is replayable

**State Management:**
- Stateless provider; all state owned by Terraform itself
- Only in-memory state inside the provider process: `samlCache` (provider.go)

## Key Abstractions

**`AxonopsHttpClient`:**
- Purpose: single point of contact with the AxonOps API
- Location: `client/http_client.go:70-98`
- Pattern: stateful struct holding `http.Client`, `protocol`, `axonopsHost`, `apiKey`, `orgid`, `tokenType`; methods build URLs from `axonops_api_version = "api/v1"` constant

**Per-resource pair (`*Resource` + `New*Resource()` constructor):**
- Purpose: implement `resource.Resource` and usually `resource.ResourceWithImportState`
- Examples: `slackIntegrationResource` in `resource_slack_integration.go:23-29`; `logAlertRuleResource` in `resource_log_alert_rule.go:80`
- Pattern: struct holds `client *axonopsClient.AxonopsHttpClient` injected via `Configure`

**Payload structs in `client/`:**
- Purpose: typed JSON shapes for both directions of the API
- 59 exported types in `client/http_client.go` (KafkaTopic, ACLResponse, MetricAlertRule, IntegrationsResponse, HealthchecksResponse, ScheduledRepairsResponse, SilenceWindow, etc.)
- Field tags use mixed `lowercase` and `camelCase` matching the AxonOps API (per project memory: integrations payloads require `integrations` field, IDs are server-generated)

## Entry Points

**Plugin entry:**
- Location: `main.go:16-32`
- Triggers: Terraform CLI launches the binary via gRPC plugin handshake
- Responsibilities: parse `--debug` flag, call `providerserver.Serve` with address `registry.terraform.io/axonops/axonops`

**Provider Configure:**
- Location: `provider.go:121-206`
- Triggers: first Terraform operation per session
- Responsibilities: validate config, decide URL pattern (SAML vs non-SAML; SaaS vs on-prem), construct `AxonopsHttpClient`, store in `resp.ResourceData`

## Architectural Constraints

- **Threading:** Terraform Plugin Framework may invoke resource methods concurrently (e.g. parallel `apply`). `AxonopsHttpClient` shares a single `*http.Client`, which is goroutine-safe. `samlCache` is guarded by `sync.RWMutex` (`provider.go:42-43`).
- **Global state:** Module-level `samlCache` and `samlCacheMu` in `provider.go`; `axonops_api_version` constant in `client/http_client.go:16` (var, but never reassigned at runtime).
- **HTTP timeout:** Hard-coded 10s in `CreateHTTPClient` (`client/http_client.go:90`). No per-call override; long-running operations (e.g. backups) rely on the API returning quickly.
- **DELETE retry budget:** `deleteMaxRetries = 3` constant (`client/delete_retry.go:14`); base backoff 1s, swappable as `var` only for tests.
- **No transitive imports:** `main` package files must not import each other beyond what Go allows in the same package; `client` must not import anything Terraform-Framework-related.

## Anti-Patterns

### Trusting the API-supplied alert rule ID

**What happens:** Naively storing `rule.ID` from the POST response into Terraform state.
**Why it's wrong:** AxonOps regenerates the ID server-side; if the client retries after a network blip the second POST creates a duplicate or the stored ID becomes stale.
**Do this instead:** Use `deterministicAlertRuleID(...)` and reconcile via `findAlertRuleByName` (`alert_rule_id.go`).

### Using `http.Get` directly from a resource file

**What happens:** A resource bypasses `AxonopsHttpClient` to issue an ad-hoc HTTP call.
**Why it's wrong:** Skips auth header injection, debug logging, TLS config, and DELETE retry. Breaks the `client` boundary.
**Do this instead:** Add a method on `AxonopsHttpClient` in `client/http_client.go` and call it. SAML probe in `provider.go:detectSAML` is the documented exception.

### Putting `org_id` after `/api/v1/` in URLs

**What happens:** `…/api/v1/something/{org_id}/…` — wrong order.
**Why it's wrong:** AxonOps returns a misleading 500. The org segment must come *between* host and `/api/v1/` in some endpoints, *after* in others — see project memory.
**Do this instead:** Follow the existing URL templates in `client/http_client.go`; they already encode the correct order per endpoint family.

## Error Handling

**Strategy:** Errors propagate as wrapped `error` values from `client` to resource layer; resources convert them into `resp.Diagnostics.AddError(...)` for Terraform.

**Patterns:**
- `client` returns `fmt.Errorf("failed to ...: %w", err)` with operation context
- Non-2xx responses become errors that include status code, URL, and response body
- 404 on DELETE is *not* an error (`client/delete_retry.go:isDeleteSuccess`)
- Validation errors use `resp.Diagnostics.AddAttributeError(path.Root("..."), ...)` (e.g. `provider.go:177`)

## Cross-Cutting Concerns

**Logging:** `tflog` for normal logs; `AXONOPS_DEBUG` env var enables raw HTTP request/response dumps with masked Authorization header (`client/http_client.go:18-68`).

**Validation:** Per-attribute via Plugin Framework schema (Required/Optional/Computed, validators, plan modifiers like `RequiresReplace`).

**Authentication:** Single static `Authorization: {tokenType} {apiKey}` header injected by every method in `client/http_client.go`.

---

*Architecture analysis: 2026-05-14*
