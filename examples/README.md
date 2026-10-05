# AxonOps Terraform Provider Examples

This directory contains example Terraform configurations for the AxonOps Terraform provider.

## Files

| File | Description |
|------|-------------|
| [provider.tf](provider.tf) | Provider configuration example |
| [topics.tf](topics.tf) | Kafka topic examples |
| [acls.tf](acls.tf) | Kafka ACL examples |
| [connectors.tf](connectors.tf) | Kafka Connect connector examples |
| [schemas.tf](schemas.tf) | Schema Registry examples (Avro, JSON Schema, Protobuf) |
| [logcollectors.tf](logcollectors.tf) | Log collector configuration examples |
| [healthchecks.tf](healthchecks.tf) | TCP, HTTP, and shell healthcheck examples |
| [cassandra_adaptive_repair.tf](cassandra_adaptive_repair.tf) | Cassandra adaptive repair configuration examples |
| [cassandra_scheduled_repair.tf](cassandra_scheduled_repair.tf) | Cassandra scheduled repair configuration examples |
| [cassandra_backups.tf](cassandra_backups.tf) | Cassandra backup configuration examples |
| [integrations.tf](integrations.tf) | Alerting integration examples (Slack, Teams, PagerDuty, OpsGenie, ServiceNow) |
| [alert_routes.tf](alert_routes.tf) | Alert route configuration examples |
| [metric_alerts.tf](metric_alerts.tf) | Metric alert rule examples |
| [silence.tf](silence.tf) | Silence window examples |
| [complete-setup.tf](complete-setup.tf) | Complete example combining multiple resource types |

## Usage

1. Copy the desired example files to your Terraform project
2. Update `provider.tf` with your AxonOps credentials
3. Modify the resource configurations as needed
4. Run Terraform:

```bash
terraform init
terraform plan
terraform apply
```

## Resource Types

### Kafka

- **axonops_kafka_topic** - Manages Kafka topics with configurations like partitions, replication factor, and topic-level settings.
- **axonops_kafka_acl** - Manages Kafka Access Control Lists for authorization.
- **axonops_kafka_connect_connector** - Manages Kafka Connect connectors (source and sink).
- **axonops_schema** - Manages schemas in Schema Registry (supports AVRO, JSON, and PROTOBUF).
- **axonops_schema_registry_compatibility** - Sets Schema Registry compatibility levels per subject or globally.

### Cassandra

- **axonops_cassandra_adaptive_repair** - Configures adaptive repair settings for Cassandra clusters.
- **axonops_cassandra_scheduled_repair** - Configures scheduled repair jobs for Cassandra clusters.
- **axonops_cassandra_backup** - Manages Cassandra backup configuration.

### Observability

- **axonops_logcollector** - Configures log collection for monitoring.
- **axonops_healthcheck_tcp** - TCP connectivity healthchecks.
- **axonops_healthcheck_http** - HTTP endpoint healthchecks.
- **axonops_healthcheck_shell** - Shell script healthchecks.
- **axonops_metric_alert_rule** - Metric-based alert rules.
- **axonops_log_alert_rule** - Log-based alert rules.
- **axonops_custom_dashboard** - Custom dashboards with panels, layout and filters.

### Access

- **axonops_api_token** - API tokens with scoped roles and rotation.
- **axonops_alert_route** - Alert routing configuration.

### Integrations

- **axonops_slack_integration** - Slack alerting integration.
- **axonops_teams_integration** - Microsoft Teams alerting integration.
- **axonops_pagerduty_integration** - PagerDuty alerting integration.
- **axonops_opsgenie_integration** - OpsGenie alerting integration.
- **axonops_servicenow_integration** - ServiceNow alerting integration.

## Notes

- Replace placeholder values (e.g., `my-kafka-cluster`, `my-cassandra-cluster`) with actual values
- Topic `partitions` can be increased in place; decreasing fails at `terraform plan`, since Kafka does not support removing partitions. `replication_factor` changes are applied in place via partition reassignment. `name` and `cluster_name` each force replacement.
- Config keys in topics use underscores in Terraform (converted to dots for Kafka API)
- Scheduled repair and backup updates are performed as delete-then-create since the API does not support in-place updates; both resources get a new `id` on every update
- Integration secrets (`webhook_url`, `integration_key`, `opsgenie_key`, `password`) are masked by the AxonOps API and are never imported — set them explicitly in configuration after `terraform import`
- `axonops_kafka_connect_connector.config` is `Sensitive` and may contain credentials embedded in the connector config; store state in an encrypted, access-controlled backend
