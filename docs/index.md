---
page_title: "AxonOps Provider"
subcategory: ""
description: |-
  The AxonOps provider enables Terraform to manage Apache Cassandra, Apache Kafka, and DSE infrastructure through AxonOps.
---

# AxonOps Provider

The AxonOps provider allows you to manage your Apache Cassandra, Apache Kafka, and DataStax Enterprise (DSE) clusters through [AxonOps](https://axonops.com). This provider supports both AxonOps SaaS and self-hosted deployments.

## Features

This provider supports managing:

### Kafka Resources
- **Topics** - Create, configure, and manage Kafka topics with partitions, replication, and custom configurations
- **ACLs** - Manage Kafka Access Control Lists for secure topic access
- **Connectors** - Deploy and manage Kafka Connect connectors
- **Schemas** - Manage Schema Registry schemas (AVRO, PROTOBUF, JSON)

### Cassandra/DSE Resources
- **Adaptive Repair** - Configure Cassandra adaptive repair settings
- **Scheduled Repair** - Schedule Cassandra repair jobs by keyspace, table, or datacenter
- **Backups** - Schedule and manage Cassandra backups with datacenter-level granularity

### Monitoring & Alerting
- **Metric Alert Rules** - Create dashboard-linked metric alerts with thresholds and filters
- **Log Alert Rules** - Create log-based alerts with content matching and severity thresholds
- **Alert Routes** - Route alerts to integrations (Slack, PagerDuty, email, Teams, ServiceNow, webhook, OpsGenie)
- **Silences** - Suppress alerts for a cluster during maintenance windows, one-off or recurring

### Integrations
- **Slack** - Configure Slack incoming webhooks for alert delivery
- **Microsoft Teams** - Configure Teams incoming webhooks for alert delivery
- **PagerDuty** - Configure PagerDuty Events API v2 for incident creation
- **OpsGenie** - Configure OpsGenie API for alert creation
- **ServiceNow** - Configure ServiceNow credentials for incident creation

### Health Checks
- **TCP Health Checks** - Monitor TCP connectivity
- **HTTP Health Checks** - Monitor HTTP endpoints
- **Shell Health Checks** - Run custom shell scripts for health monitoring

### Log Collection
- **Log Collectors** - Configure log collection settings

## Authentication

The provider supports API key authentication. You can configure credentials via provider configuration or environment variables.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `AXONOPS_API_KEY` | API key for authentication |
| `AXONOPS_HOST` | AxonOps server hostname |
| `AXONOPS_PROTOCOL` | Protocol (http/https) |
| `AXONOPS_TOKEN_TYPE` | Token type: 'Bearer' or 'AxonApi' |
| `AXONOPS_TLS_SKIP_VERIFY` | Skip TLS certificate verification |

## SAML Authentication

The provider has no `use_saml` attribute or environment variable — SAML is detected automatically. On every `Configure` call, the provider probes `{host}/dashboard` and switches URL routing if the probe indicates a SAML-enabled organization:

- SAML organization, no `axonops_host` set: `https://{org_id}.axonops.cloud/dashboard`
- Non-SAML organization, no `axonops_host` set: `https://dash.axonops.cloud/{org_id}`
- SAML organization, custom `axonops_host`: `https://{axonops_host}/dashboard`
- Non-SAML organization, custom `axonops_host`: `https://{axonops_host}`

No provider configuration is required to use SAML; simply point `axonops_host` (or leave it unset for SaaS) at a SAML-enabled organization and the provider adapts automatically.

## Example Usage

### SaaS Deployment (SAML Auto-Detected)

```terraform
terraform {
  required_providers {
    axonops = {
      source  = "axonops/axonops"
      version = "~> 0.1"
    }
  }
}

provider "axonops" {
  org_id  = var.axonops_org_id
  api_key = var.axonops_api_key
}
```

### Self-Hosted Deployment

For self-hosted AxonOps deployments, specify the server hostname. SAML is still auto-detected:

```terraform
provider "axonops" {
  org_id           = var.axonops_org_id
  api_key          = var.axonops_api_key
  axonops_host     = "axonops.example.com"
  axonops_protocol = "https"
  token_type       = "Bearer"
  tls_skip_verify  = false  # Only for self-signed certificates in non-production
}
```

### Environment Variables

Avoid hardcoding credentials. Use environment variables instead:

```bash
export AXONOPS_ORG_ID="your-org-id"
export AXONOPS_API_KEY="your-api-key"
export AXONOPS_PROTOCOL="https"
```

Then configure the provider without credentials:

```terraform
provider "axonops" {
  org_id = var.axonops_org_id
  # api_key will be read from AXONOPS_API_KEY environment variable
}
```

## Resources

### Kafka Management
- `axonops_kafka_topic` - Manage Kafka topics
- `axonops_kafka_acl` - Manage Kafka ACLs
- `axonops_kafka_connect_connector` - Deploy Kafka Connect connectors

### Cassandra/DSE Management
- `axonops_cassandra_backup` - Schedule and manage backups
- `axonops_cassandra_scheduled_repair` - Schedule repair jobs
- `axonops_cassandra_adaptive_repair` - Configure adaptive repair

### Monitoring & Alerting
- `axonops_metric_alert_rule` - Create metric-based alert rules
- `axonops_log_alert_rule` - Create log-based alert rules
- `axonops_alert_route` - Route alerts to integrations
- `axonops_silence` - Suppress alerts during maintenance

### Integrations
- `axonops_slack_integration` - Configure Slack webhook
- `axonops_teams_integration` - Configure Microsoft Teams webhook
- `axonops_pagerduty_integration` - Configure PagerDuty Events API
- `axonops_opsgenie_integration` - Configure OpsGenie API
- `axonops_servicenow_integration` - Configure ServiceNow credentials

### Health Checks & Log Collection
- `axonops_healthcheck_tcp` - Monitor TCP connectivity
- `axonops_healthcheck_http` - Monitor HTTP endpoints
- `axonops_healthcheck_shell` - Run custom shell health checks
- `axonops_logcollector` - Configure log collection
- `axonops_schema` - Manage Schema Registry schemas

## Data Sources

### Kafka
- `axonops_kafka_topic` - Look up a Kafka topic
- `axonops_kafka_acl` - Look up a specific Kafka ACL
- `axonops_kafka_acl_list` - List all Kafka ACLs
- `axonops_kafka_connect_connector` - Look up a Kafka Connect connector

### Cassandra/DSE
- `axonops_cassandra_backup` - Look up a backup
- `axonops_cassandra_scheduled_repair` - Look up a repair schedule
- `axonops_cassandra_adaptive_repair` - Look up adaptive repair settings

### Monitoring & Alerting
- `axonops_metric_alert_rule` - Look up a metric alert rule
- `axonops_log_alert_rule` - Look up a log alert rule
- `axonops_alert_route` - Look up an alert route
- `axonops_silence` - Look up a silence

### Integrations
- `axonops_slack_integration` - Look up a Slack integration
- `axonops_teams_integration` - Look up a Teams integration
- `axonops_pagerduty_integration` - Look up a PagerDuty integration
- `axonops_opsgenie_integration` - Look up an OpsGenie integration
- `axonops_servicenow_integration` - Look up a ServiceNow integration

### Health Checks & Log Collection
- `axonops_healthcheck_tcp` - Look up a TCP health check
- `axonops_healthcheck_http` - Look up an HTTP health check
- `axonops_healthcheck_shell` - Look up a shell health check
- `axonops_logcollector` - Look up a log collector
- `axonops_schema` - Look up a Schema Registry schema

## Security

- `api_key` is `Sensitive` — Terraform redacts it from plan/apply output and `terraform show`, but it is still stored in plaintext in the state file. Use an encrypted, access-controlled state backend.
- `tls_skip_verify` MUST NOT be set to `true` against a production AxonOps server. It disables TLS certificate verification entirely, which allows a network-level attacker to intercept `api_key` and all provider traffic via a man-in-the-middle attack. When set to `true`, the provider emits a `TLS Certificate Verification Disabled` warning on every plan and apply as a standing reminder. Reserve it for test environments using self-signed certificates.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `org_id` (String) The AxonOps organization ID. This identifies your organization within AxonOps and is required for authentication.

### Optional

- `api_key` (String, Sensitive) The API key for authentication with AxonOps. Generate this from your AxonOps dashboard. If not provided, the provider will use the AXONOPS_API_KEY environment variable. This value is sensitive and should be stored securely.
- `axonops_host` (String) The AxonOps server hostname without the protocol (e.g., 'axonops.example.com' or 'myorg.axonops.cloud'). For AxonOps SaaS, leave this empty to auto-detect the correct URL based on org_id and SAML configuration. For self-hosted deployments, specify your server's fully qualified domain name. Default: Auto-detected for SaaS. Environment variable: AXONOPS_HOST.
- `axonops_protocol` (String) The protocol to use when connecting to the AxonOps API. Valid values: 'https' (default, recommended for production) or 'http' (only for non-production environments). Default: 'https'. Environment variable: AXONOPS_PROTOCOL.
- `tls_skip_verify` (Boolean) Skip TLS certificate verification when connecting to the AxonOps API. Use only for development environments with self-signed certificates. WARNING: Disabling TLS verification exposes your API key to man-in-the-middle attacks. Default: false. Environment variable: AXONOPS_TLS_SKIP_VERIFY (set to 'true' to enable).
- `token_type` (String) The type of authentication token to use in the Authorization header. Valid values: 'Bearer' (default for SaaS) or 'AxonApi' (typically for on-premise deployments). Most users should leave this at the default. Default: 'Bearer'. Environment variable: AXONOPS_TOKEN_TYPE.
