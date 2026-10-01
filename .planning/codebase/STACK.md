# Technology Stack

**Analysis Date:** 2026-05-14

## Languages

**Primary:**
- Go 1.24.0 (toolchain go1.24.10) — entire provider implementation (`go.mod`)

**Secondary:**
- HCL — Terraform configuration in `examples/`
- Python 3 — single helper script `scripts/import-cluster.py`
- Markdown templates (`templates/*.tmpl`) — auto-generated documentation source

## Runtime

**Environment:**
- Go 1.23 and 1.24 tested in CI (`.github/workflows/test.yml`)
- Built as a Terraform provider plugin invoked by Terraform CLI / OpenTofu
- gRPC plugin protocol via `github.com/hashicorp/terraform-plugin-go` v0.24.0

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- `github.com/hashicorp/terraform-plugin-framework` v1.12.0 — modern Plugin Framework (NOT the legacy Plugin SDK v2). Provides `provider.Provider`, `resource.Resource`, `datasource.DataSource` interfaces.
- `github.com/hashicorp/terraform-plugin-log` v0.9.0 — `tflog` structured logging surfaced via `terraform plan/apply -tf-log`

**Testing:**
- Go standard `testing` package — used in `client/delete_retry_test.go`, `alert_rule_id_test.go`, `resource_log_alert_rule_test.go`
- `TF_ACC=1` acceptance test gate (`Makefile` `testacc` target)
- No third-party assertion library detected; tests use `t.Fatalf` / `t.Errorf`

**Build/Dev:**
- `goreleaser` v2 — multi-OS / multi-arch release builds (`.goreleaser.yml`)
- `golangci-lint` — invoked via `make lint`
- `gofmt -s -w .` via `make fmt`
- `go generate` runs `github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs` to regenerate `docs/` from `templates/` (see `main.go` `//go:generate` directive)

## Key Dependencies

**Critical:**
- `github.com/hashicorp/terraform-plugin-framework` v1.12.0 — provider runtime
- `github.com/hashicorp/terraform-plugin-go` v0.24.0 (indirect) — wire protocol
- `github.com/google/uuid` v1.6.0 (indirect, used directly in `alert_rule_id.go`) — UUID v5 generation for deterministic alert IDs
- `github.com/hashicorp/terraform-plugin-docs` v0.24.0 (indirect) — docs generation tool

**Infrastructure:**
- Standard library only for HTTP transport (`net/http`, `crypto/tls`, `encoding/json`)
- No external HTTP client or retry library — custom retry logic in `client/delete_retry.go`

## Configuration

**Environment:**
- `AXONOPS_HOST` — server hostname (no protocol)
- `AXONOPS_PROTOCOL` — `https` (default) or `http`
- `AXONOPS_API_KEY` — bearer / api token
- `AXONOPS_TOKEN_TYPE` — `Bearer` (default, SaaS) or `AxonApi` (on-prem)
- `AXONOPS_TLS_SKIP_VERIFY` — `true` to skip TLS verification
- `AXONOPS_DEBUG` — non-empty enables verbose request/response logging in `client/http_client.go` and SAML probe in `provider.go`
- `TF_ACC=1` — enables acceptance tests

All env vars overridable per-provider-block via attributes defined in `provider.go` `Schema()`.

**Build:**
- `Makefile` — `build`, `test`, `testacc`, `lint`, `docs`, `fmt`, `buildnrun`
- `.goreleaser.yml` — release pipeline targeting freebsd/windows/linux/darwin × amd64/386/arm/arm64
- `terraform-registry-manifest.json` — Terraform Registry metadata

## Platform Requirements

**Development:**
- Go 1.24+
- Terraform CLI (or OpenTofu) for local `make buildnrun`
- GPG key (only for releases — used by `crazy-max/ghaction-import-gpg`)

**Production:**
- Distributed via Terraform Registry as `axonops/axonops` (`main.go`: `Address: "registry.terraform.io/axonops/axonops"`)
- Binary name pattern: `terraform-provider-axonops_v{version}` per `.goreleaser.yml`
- Released to GitHub Releases on `v*` tag push (`.github/workflows/release.yml`)

---

*Stack analysis: 2026-05-14*
