# Codebase Structure

**Analysis Date:** 2026-05-14

## Directory Layout

```
terraform-provider-axonops/
├── main.go                          # Plugin entry, providerserver.Serve
├── provider.go                      # axonopsProvider, Schema, Configure, SAML detection
├── alert_rule_id.go                 # Deterministic UUID v5 + name-based reconciliation
├── alert_rule_id_test.go
├── resource_*.go                    # 20 Terraform resource implementations
├── resource_log_alert_rule_test.go  # Only resource-level test currently in repo
├── data_source_*.go                 # 19 Terraform data source implementations
├── client/                          # AxonOps HTTP API client (single subpackage)
│   ├── http_client.go               # ~2100 lines, AxonopsHttpClient + 59 payload structs
│   ├── delete_retry.go              # Idempotent DELETE with exponential backoff
│   └── delete_retry_test.go
├── docs/                            # Auto-generated from templates/ via tfplugindocs
│   ├── resources/                   # 20 resource docs
│   └── data-sources/                # data source docs
├── templates/                       # Source templates for docs generation
│   ├── index.md.tmpl
│   ├── resources/
│   └── data-sources/
├── examples/                        # HCL examples published to Terraform Registry
│   ├── provider/
│   ├── resources/{axonops_alert_route,axonops_log_alert_rule,axonops_metric_alert_rule}/
│   ├── data-sources/axonops_log_alert_rule/
│   ├── cassandra/                   # End-to-end Cassandra example
│   └── kafka/                       # End-to-end Kafka example
├── scripts/
│   └── import-cluster.py            # Python helper for bulk import
├── .github/workflows/               # 6 CI workflows
├── .goreleaser.yml                  # Multi-OS/arch release pipeline
├── Makefile                         # build, test, testacc, lint, docs, fmt
├── go.mod / go.sum
├── terraform-registry-manifest.json
├── README.md
├── LICENSE                          # Apache-2.0
└── CLAUDE.md                        # Agent workflow instructions for this repo
```

## Directory Purposes

**`/` (root, package `main`):**
- Purpose: provider runtime + every resource and data source
- Contains: `provider.go`, one `resource_*.go` per resource, one `data_source_*.go` per data source, `alert_rule_id.go` helpers
- Key files: `main.go`, `provider.go`, `alert_rule_id.go`

**`client/`:**
- Purpose: only place that talks HTTP to AxonOps
- Contains: `AxonopsHttpClient`, all payload types, retry helpers
- Key files: `client/http_client.go`, `client/delete_retry.go`

**`docs/`:**
- Purpose: published documentation rendered onto Terraform Registry
- Generated: yes — by `make docs` (`go generate ./...` → `tfplugindocs`)
- Committed: yes (kept in sync via `.github/workflows/docs.yml`)

**`templates/`:**
- Purpose: source-of-truth markdown templates for `docs/`
- Edit here, never edit `docs/*.md` directly

**`examples/`:**
- Purpose: HCL snippets shown on the Terraform Registry page for each resource and end-to-end deployment examples
- Subdirectory naming under `resources/` and `data-sources/` MUST be `axonops_<resource_name>/` for `tfplugindocs` to pick them up

**`scripts/`:**
- Purpose: out-of-band helpers (currently one Python cluster import script)

**`models_cache/`, `data/chroma_db/`, `documents/`:**
- Purpose: local AI/embedding artefacts unrelated to provider runtime
- Generated: yes — present from `.gitignore` exclusions; not part of release builds
- Committed: appears to be gitignored or local-only

**`.planning/`:**
- Purpose: GSD planning artefacts; this codebase map lives here

## Key File Locations

**Entry Points:**
- `main.go`: gRPC plugin entry, calls `providerserver.Serve` with registry address
- `provider.go`: provider Schema/Configure/Resources/DataSources

**Configuration:**
- `go.mod`: dependencies and Go version
- `Makefile`: developer task runner
- `.goreleaser.yml`: release matrix
- `terraform-registry-manifest.json`: registry metadata
- `.github/workflows/*.yml`: CI

**Core Logic:**
- `client/http_client.go`: API contract surface (URLs, methods, payload structs)
- `client/delete_retry.go`: DELETE retry policy
- `alert_rule_id.go`: deterministic ID + name-based reconciliation pattern
- `resource_metric_alert_rule.go` (32 KB) and `resource_log_alert_rule.go` (19 KB): largest, most complex resources — reference implementations

**Testing:**
- `client/delete_retry_test.go`
- `alert_rule_id_test.go`
- `resource_log_alert_rule_test.go`

## Naming Conventions

**Files:**
- Resources: `resource_<terraform_type_without_axonops_prefix>.go` — e.g. resource type `axonops_kafka_topic` → `resource_kafka_topic.go`
- Data sources: `data_source_<terraform_type_without_axonops_prefix>.go`
- Tests: `<file>_test.go` co-located with the code they test
- Docs templates: `templates/resources/<resource_name>.md.tmpl`
- Examples: `examples/resources/axonops_<resource_name>/resource.tf`

**Directories:**
- `examples/resources/axonops_*` (full provider-prefixed name) — required by `tfplugindocs`
- `docs/resources/<resource_name>.md` (no prefix)

**Go identifiers:**
- Resource struct: `<lowerCamel>Resource` (e.g. `slackIntegrationResource`)
- Resource constructor: `New<UpperCamel>Resource` (e.g. `NewSlackIntegrationResource`) — registered in `provider.go` `Resources()`
- Data source struct/constructor mirror: `*DataSource` / `New*DataSource`
- Client methods: verb-noun PascalCase (`CreateTopic`, `GetIntegrations`, `DeleteAlertRule`)

## Where to Add New Code

**New Terraform resource (e.g. `axonops_foo_bar`):**
- Implementation: `resource_foo_bar.go` at repo root, package `main`
- Constructor: `NewFooBarResource()` returning `resource.Resource`
- Registration: append to slice in `provider.go` `Resources()` (`provider.go:236-259`)
- API methods: add to `client/http_client.go` as methods on `*AxonopsHttpClient`
- Payload structs: add to `client/http_client.go` near related types
- Docs template: `templates/resources/foo_bar.md.tmpl`
- Example: `examples/resources/axonops_foo_bar/resource.tf`
- Regenerate docs: `make docs`

**New Terraform data source:**
- Same as above but `data_source_foo_bar.go`, register in `provider.go` `DataSources()` (`provider.go:212-234`)

**New API endpoint (no Terraform surface yet):**
- Add method on `*AxonopsHttpClient` in `client/http_client.go`
- Add corresponding request/response structs alongside it
- For DELETE endpoints, route through `c.doDeleteWithRetry(req, body)` not `c.client.Do(req)` directly

**Shared helpers used by multiple resources:**
- If purely client-facing → `client/` package
- If Terraform-schema-related → root package, named after the concern (see `alert_rule_id.go` precedent: small focused file, dedicated test)

**Tests:**
- Unit test: `<file>_test.go` next to the code
- Acceptance test: same file with `TF_ACC` guard; runs under `make testacc`

## Special Directories

**`models_cache/`:**
- Purpose: HuggingFace model cache (unrelated to provider runtime)
- Generated: yes
- Committed: should be gitignored

**`data/chroma_db/`:**
- Purpose: local Chroma vector DB (unrelated to provider runtime)
- Generated: yes
- Committed: should be gitignored

**`documents/`:**
- Purpose: local notes / artefacts directory
- Not part of the published provider

**`docs/`:**
- Purpose: published documentation
- Generated: yes (from `templates/`)
- Committed: yes — `docs.yml` workflow enforces it stays in sync

---

*Structure analysis: 2026-05-14*
