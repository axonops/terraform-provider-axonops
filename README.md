<div align="center">

<img src="https://digitalis-marketplace-assets.s3.us-east-1.amazonaws.com/AxonopsDigitalMaster_AxonopsFullLogoBlue.jpg" alt="AxonOps" width="300">

# AxonOps Terraform Provider

**Infrastructure as Code for Apache Cassandra, Apache Kafka, and DSE clusters managed through [AxonOps](https://axonops.com)**

[![Tests](https://github.com/axonops/terraform-provider-axonops/actions/workflows/test.yml/badge.svg)](https://github.com/axonops/terraform-provider-axonops/actions/workflows/test.yml)
[![Documentation](https://img.shields.io/badge/docs-registry.terraform.io-blue)](https://registry.terraform.io/providers/axonops/axonops/latest/docs)
[![License](https://img.shields.io/github/license/axonops/terraform-provider-axonops)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/axonops/terraform-provider-axonops)](go.mod)

</div>

A Terraform provider for managing resources through the AxonOps platform: Kafka topics, ACLs, connectors, and schemas; Cassandra backups and repairs; health checks, log collectors, metric and log alert rules, alert routes, silences, and alerting integrations (Slack, Microsoft Teams, PagerDuty, OpsGenie, ServiceNow).

## Quick Start

```hcl
terraform {
  required_providers {
    axonops = {
      source = "axonops/axonops"
    }
  }
}

provider "axonops" {
  org_id  = "my-organization"    # Required
  api_key = var.axonops_api_key  # Required for AxonOps SaaS
}

resource "axonops_kafka_topic" "events" {
  name               = "user-events"
  partitions         = 6
  replication_factor = 3
  cluster_name       = "production-kafka"

  config = {
    retention_ms   = "604800000"
    cleanup_policy = "delete"
  }
}
```

```bash
terraform init
terraform plan
terraform apply
```

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.27.1 (for building from source)
- Access to an AxonOps instance (SaaS or self-hosted)

## Installation

### From the Terraform Registry

```hcl
terraform {
  required_providers {
    axonops = {
      source = "axonops/axonops"
    }
  }
}
```

```bash
terraform init
```

### Building from Source

```bash
git clone https://github.com/axonops/terraform-provider-axonops.git
cd terraform-provider-axonops
go build -o terraform-provider-axonops
```

### Development Override

To point Terraform at a locally built binary instead of the registry, add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "axonops/axonops" = "/path/to/terraform-provider-axonops"
  }
  direct {}
}
```

## Features

- **Kafka** — topics, ACLs, Kafka Connect connectors, Schema Registry schemas (AVRO, Protobuf, JSON)
- **Cassandra/DSE** — adaptive repair, scheduled repair, backups
- **Health checks** — TCP, HTTP, and shell health checks
- **Log collection** — log collector configuration (on-prem only)
- **Alerting** — metric alert rules, log alert rules, alert routes, silences
- **Integrations** — Slack, Microsoft Teams, PagerDuty, OpsGenie, ServiceNow

## Configuration Reference

```hcl
provider "axonops" {
  org_id           = "your-org-id"          # Required
  api_key          = var.axonops_api_key    # Required for AxonOps SaaS
  axonops_host     = "axonops.example.com"  # Optional, defaults to auto-detected SaaS routing
  axonops_protocol = "https"                # Optional
  token_type       = "Bearer"               # Optional
  tls_skip_verify  = false                  # Optional
}
```

| Attribute | Type | Required | Default | Environment Variable | Example |
|-----------|------|----------|---------|----------------------|---------|
| `org_id` | string | Yes | — | — | `"my-organization"` |
| `api_key` | string, sensitive | Required for SaaS | — | `AXONOPS_API_KEY` | `var.axonops_api_key` |
| `axonops_host` | string | No | Auto-detected: `dash.axonops.cloud/<org_id>` (SaaS) or `<org_id>.axonops.cloud/dashboard` (SAML) | `AXONOPS_HOST` | `"axonops.example.com"` |
| `axonops_protocol` | string | No | `https` | `AXONOPS_PROTOCOL` | `"https"` |
| `token_type` | string | No | `Bearer` | `AXONOPS_TOKEN_TYPE` | `"AxonApi"` (on-premise) |
| `tls_skip_verify` | bool | No | `false` | `AXONOPS_TLS_SKIP_VERIFY` | `false` |

> **Warning:** `tls_skip_verify` MUST NOT be set to `true` against a production AxonOps server. It disables TLS certificate verification, exposing `api_key` and all provider traffic to man-in-the-middle interception. The provider emits a `TLS Certificate Verification Disabled` warning on every plan/apply while it is enabled. Reserve it for test environments with self-signed certificates.

A statically configured attribute always overrides its environment variable. SAML organizations are detected automatically by probing `{host}/dashboard` — there is no `use_saml` attribute to set.

Full attribute documentation: [registry.terraform.io/providers/axonops/axonops/latest/docs](https://registry.terraform.io/providers/axonops/axonops/latest/docs).

## Resources

| Resource | Description |
|----------|--------------|
| [`axonops_kafka_topic`](docs/resources/kafka_topic.md) | Kafka topic: partitions, replication factor, topic config |
| [`axonops_kafka_acl`](docs/resources/kafka_acl.md) | Kafka Access Control List entry |
| [`axonops_kafka_connect_connector`](docs/resources/kafka_connect_connector.md) | Kafka Connect source/sink connector |
| [`axonops_schema`](docs/resources/schema.md) | Schema Registry schema (AVRO, Protobuf, JSON) |
| [`axonops_cassandra_adaptive_repair`](docs/resources/cassandra_adaptive_repair.md) | Cassandra adaptive repair settings |
| [`axonops_cassandra_scheduled_repair`](docs/resources/cassandra_scheduled_repair.md) | Cassandra scheduled repair job |
| [`axonops_cassandra_backup`](docs/resources/cassandra_backup.md) | Cassandra backup schedule |
| [`axonops_healthcheck_tcp`](docs/resources/healthcheck_tcp.md) | TCP connectivity health check |
| [`axonops_healthcheck_http`](docs/resources/healthcheck_http.md) | HTTP endpoint health check |
| [`axonops_healthcheck_shell`](docs/resources/healthcheck_shell.md) | Shell script health check |
| [`axonops_logcollector`](docs/resources/logcollector.md) | Log collector configuration (on-prem only) |
| [`axonops_metric_alert_rule`](docs/resources/metric_alert_rule.md) | Dashboard-linked metric alert rule |
| [`axonops_log_alert_rule`](docs/resources/log_alert_rule.md) | Log content-based alert rule |
| [`axonops_alert_route`](docs/resources/alert_route.md) | Route alerts to an integration |
| [`axonops_silence`](docs/resources/silence.md) | Silence window (one-off or recurring) |
| [`axonops_slack_integration`](docs/resources/slack_integration.md) | Slack incoming webhook integration |
| [`axonops_teams_integration`](docs/resources/teams_integration.md) | Microsoft Teams incoming webhook integration |
| [`axonops_pagerduty_integration`](docs/resources/pagerduty_integration.md) | PagerDuty Events API v2 integration |
| [`axonops_opsgenie_integration`](docs/resources/opsgenie_integration.md) | OpsGenie alerting integration |
| [`axonops_servicenow_integration`](docs/resources/servicenow_integration.md) | ServiceNow incident integration |

## Data Sources

| Data Source | Description |
|-------------|--------------|
| [`axonops_clusters`](docs/data-sources/clusters.md) | List clusters in the organisation, optionally by type |
| [`axonops_cluster`](docs/data-sources/cluster.md) | Look up one cluster by name (type, status, node count, versions) |
| [`axonops_cluster_nodes`](docs/data-sources/cluster_nodes.md) | List nodes of a cluster, filter by status or data centre |
| [`axonops_cassandra_keyspaces`](docs/data-sources/cassandra_keyspaces.md) | List keyspaces and tables of a Cassandra/DSE cluster |
| [`axonops_kafka_topics`](docs/data-sources/kafka_topics.md) | List topics of a Kafka cluster |
| [`axonops_kafka_connectors`](docs/data-sources/kafka_connectors.md) | List connectors of a Kafka Connect cluster, filter by type |
| [`axonops_schemas`](docs/data-sources/schemas.md) | List Schema Registry subjects, filter by regex |
| [`axonops_integrations`](docs/data-sources/integrations.md) | List alert integrations of a cluster, filter by type (no secrets) |
| [`axonops_kafka_topic`](docs/data-sources/kafka_topic.md) | Read an existing Kafka topic |
| [`axonops_kafka_acl`](docs/data-sources/kafka_acl.md) | Read a single Kafka ACL matching exact identity fields |
| [`axonops_kafka_acl_list`](docs/data-sources/kafka_acl_list.md) | List Kafka ACLs matching partial criteria |
| [`axonops_kafka_connect_connector`](docs/data-sources/kafka_connect_connector.md) | Read an existing Kafka Connect connector |
| [`axonops_schema`](docs/data-sources/schema.md) | Read an existing Schema Registry schema |
| [`axonops_cassandra_adaptive_repair`](docs/data-sources/cassandra_adaptive_repair.md) | Read adaptive repair settings |
| [`axonops_cassandra_scheduled_repair`](docs/data-sources/cassandra_scheduled_repair.md) | Read a scheduled repair job |
| [`axonops_cassandra_backup`](docs/data-sources/cassandra_backup.md) | Read a backup schedule |
| [`axonops_healthcheck_tcp`](docs/data-sources/healthcheck_tcp.md) | Read a TCP health check |
| [`axonops_healthcheck_http`](docs/data-sources/healthcheck_http.md) | Read an HTTP health check |
| [`axonops_healthcheck_shell`](docs/data-sources/healthcheck_shell.md) | Read a shell health check |
| [`axonops_logcollector`](docs/data-sources/logcollector.md) | Read a log collector configuration |
| [`axonops_metric_alert_rule`](docs/data-sources/metric_alert_rule.md) | Read a metric alert rule |
| [`axonops_log_alert_rule`](docs/data-sources/log_alert_rule.md) | Read a log alert rule |
| [`axonops_alert_route`](docs/data-sources/alert_route.md) | Read an alert route |
| [`axonops_silence`](docs/data-sources/silence.md) | Read a silence window |
| [`axonops_slack_integration`](docs/data-sources/slack_integration.md) | Read a Slack integration (secret returned as `null`) |
| [`axonops_teams_integration`](docs/data-sources/teams_integration.md) | Read a Teams integration (secret returned as `null`) |
| [`axonops_pagerduty_integration`](docs/data-sources/pagerduty_integration.md) | Read a PagerDuty integration (secret returned as `null`) |
| [`axonops_opsgenie_integration`](docs/data-sources/opsgenie_integration.md) | Read an OpsGenie integration (secret returned as `null`) |
| [`axonops_servicenow_integration`](docs/data-sources/servicenow_integration.md) | Read a ServiceNow integration (secret returned as `null`) |

More usage examples, per-resource attribute tables, and import details live in [`docs/`](docs/) and [`examples/`](examples/).

## Behaviour Notes

These behaviours are not obvious from the attribute tables alone and MUST be understood before relying on them in production:

- **Integration secrets are never imported.** `webhook_url`, `integration_key`, `opsgenie_key`, and `password` are masked by the AxonOps API on read, so `terraform import` leaves them unset and emits a `Sensitive Value Not Imported` warning. Set them explicitly in configuration immediately after import, or the next `terraform plan` shows the secret being applied.
- **`axonops_kafka_topic.partitions`** MAY be increased in place; decreasing it fails at `terraform plan` with `Cannot Decrease Partitions`, because Kafka does not support removing partitions. `replication_factor` changes are applied in place via partition reassignment. `name` and `cluster_name` force replacement.
- **`axonops_kafka_connect_connector.config`** is `Sensitive` and only tracks the keys present in your Terraform configuration; server-injected keys (e.g. `name`) are ignored. Because connector configs commonly embed credentials, store state in an encrypted, access-controlled backend.
- **`axonops_silence.note`** requires an AxonOps server newer than `2.0.39`. Against older servers the attribute is silently ignored server-side, producing a persistent `terraform plan` diff — omit it when targeting an older server.
- **`axonops_alert_route`** identity fields (`cluster_name`, `cluster_type`, `type`, `severity`, `integration_type`, `integration_name`) all force replacement; only `enable_override` updates in place.
- **`axonops_cassandra_backup` updates are delete-then-create.** The AxonOps API has no in-place update for backup schedules, so every update changes the resource's `id`, even for a single-attribute change.
- **Health checks and log collectors share one document per cluster.** The provider serializes writes across sibling resources on the same `cluster_type`/`cluster_name` within a single `terraform apply`, preventing intra-process lost updates. This protection does NOT extend across separate `apply` processes (e.g. two CI pipelines) or concurrent edits via the AxonOps UI — those can still race.
- **`tls_skip_verify`** MUST NOT be used against production servers — see [Configuration Reference](#configuration-reference).
- **`api_key`** is `Sensitive` in Terraform output, but is still stored in plaintext in the state file. Use an encrypted, access-controlled state backend.

## Importing Existing Resources

All resources support `terraform import`.

| Resource | Import ID Format |
|----------|-------------------|
| `axonops_kafka_topic` | `cluster_name/topic_name` |
| `axonops_kafka_acl` | `cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type` (`principal` may contain `/`) |
| `axonops_kafka_connect_connector` | `cluster_name/connect_cluster_name/connector_name` (`connector_name` may contain `/`) |
| `axonops_schema` | `cluster_name/subject` |
| `axonops_cassandra_adaptive_repair` | `cluster_type/cluster_name` |
| `axonops_cassandra_scheduled_repair` | `cluster_type/cluster_name/tag` (or legacy `cluster_name/tag`) |
| `axonops_cassandra_backup` | `cluster_type/cluster_name/tag` |
| `axonops_healthcheck_tcp` | `cluster_type/cluster_name/healthcheck_name` |
| `axonops_healthcheck_http` | `cluster_type/cluster_name/healthcheck_name` |
| `axonops_healthcheck_shell` | `cluster_type/cluster_name/healthcheck_name` |
| `axonops_logcollector` | `cluster_type/cluster_name/filename` (`filename` is the collector's `filename` attribute, not its `name`, and may itself start with `/`) |
| `axonops_metric_alert_rule` | `cluster_type/cluster_name/alert_id` |
| `axonops_log_alert_rule` | `cluster_type/cluster_name/alert_id` |
| `axonops_alert_route` | `cluster_type/cluster_name/type/severity/integration_type/integration_name` (6 parts, in this order) |
| `axonops_silence` | `cluster_type/cluster_name/silence_id` |
| `axonops_slack_integration` | `cluster_type/cluster_name/name` |
| `axonops_teams_integration` | `cluster_type/cluster_name/name` |
| `axonops_pagerduty_integration` | `cluster_type/cluster_name/name` |
| `axonops_opsgenie_integration` | `cluster_type/cluster_name/name` |
| `axonops_servicenow_integration` | `cluster_type/cluster_name/name` |

```bash
# Kafka topic
terraform import axonops_kafka_topic.events "production-kafka/user-events"

# Kafka ACL
terraform import axonops_kafka_acl.events_read "production-kafka/TOPIC/user-events/LITERAL/User:consumer-app/*/READ/ALLOW"

# Alert route (6 parts: cluster_type/cluster_name/type/severity/integration_type/integration_name)
terraform import axonops_alert_route.pagerduty_global "cassandra/production-cassandra/global/error/pagerduty/pagerduty-oncall"

# Log collector (filename starts with "/", producing a double slash after cluster_name)
terraform import axonops_logcollector.server_log "kafka/production-kafka//var/log/kafka/server.log"

# Integration (secret is left unset — set it in configuration after import)
terraform import axonops_slack_integration.ops_alerts "cassandra/production-cassandra/ops-slack-alerts"
```

### Bulk Import Script

To import an entire cluster, use the provided Python script:

```bash
python3 scripts/import-cluster.py <axonops_host> <org_id> <cluster_name> <api_key> [output_dir]

# Example
python3 scripts/import-cluster.py axonops.example.com myorg mycluster abc123 ./imported
```

The script generates `.tf` files for the cluster's resources, an `import_commands.sh` script with the matching `terraform import` commands, and a `provider.tf`.

After running it:
1. Review the generated `.tf` files in the output directory.
2. Set your API key: `export TF_VAR_axonops_api_key='your-api-key'`.
3. `terraform init`
4. `bash import_commands.sh`
5. `terraform plan` — this MUST show no changes; any diff means the generated configuration and the live resource disagree.

## Development

```bash
make build      # go build -o terraform-provider-axonops
make test       # go test -v -cover ./...
make testacc    # acceptance tests (TF_ACC=1); talks to a real or mocked AxonOps API
make lint       # golangci-lint run
make docs       # go generate ./... — regenerates docs/ from templates/ and examples/
make fmt        # gofmt -s -w .
```

`docs/` is generated from `templates/` and `examples/` via [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs) — do not hand-edit files under `docs/`. After changing a resource's schema or an example under `examples/`, run `make docs` and commit the regenerated output alongside your change.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

## Support

Maintained by [AxonOps](https://axonops.com). For support, visit [axonops.com/contact](https://axonops.com/contact).

***

*This project may contain trademarks or logos for projects, products, or services. Any use of third-party trademarks or logos are subject to those third-party's policies. AxonOps is a registered trademark of AxonOps Limited. Apache, Apache Cassandra, Cassandra, Apache Spark, Spark, Apache TinkerPop, TinkerPop, Apache Kafka and Kafka are either registered trademarks or trademarks of the Apache Software Foundation or its subsidiaries in Canada, the United States and/or other countries. Elasticsearch is a trademark of Elasticsearch B.V., registered in the U.S. and in other countries. Docker is a trademark or registered trademark of Docker, Inc. in the United States and/or other countries.*
