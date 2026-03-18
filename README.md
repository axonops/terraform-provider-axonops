# AxonOps Terraform Provider

A Terraform provider for managing resources through the AxonOps platform. This provider enables Infrastructure as Code (IaC) management of Kafka topics, ACLs, connectors, schemas, Cassandra backups, healthchecks, alerting, and more.

## Features

- **Topics**: Create, update, and delete Kafka topics with custom configurations
- **ACLs**: Manage Kafka Access Control Lists for fine-grained permissions
- **Connectors**: Deploy and manage Kafka Connect connectors
- **Schemas**: Register and version schemas in Schema Registry (AVRO, Protobuf, JSON)
- **Integrations**: Configure Slack, Microsoft Teams, PagerDuty, OpsGenie, and ServiceNow alerting integrations
- **Alert routes**: Route alerts by category and severity to any configured integration

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23 (for building from source)
- Access to an AxonOps instance

## Installation

### Building from Source

```bash
git clone https://github.com/axonops/axonops-tf.git
cd axonops-tf
go build -o terraform-provider-axonops
```

### Development Override

For local development, add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "axonops/axonops" = "/path/to/axonops-tf"
  }
  direct {}
}
```

### Install from Terraform Registry

Add the provider to your Terraform configuration and run `terraform init`:

```hcl
terraform {
  required_providers {
    axonops = {
      source = "axonops/axonops"
    }
  }
}

provider "axonops" {
  api_key = "your-api-key"  # Required for AxonOps SaaS
  org_id  = "your-org-id"   # Required
}
```

```bash
terraform init
```

## Provider Configuration

```hcl
provider "axonops" {
  api_key          = "your-api-key"        # Required for AxonOps SaaS
  axonops_host     = "axonops.example.com" # Default: dash.axonops.cloud/<org_id>
  axonops_protocol = "https"               # Default: https
  org_id           = "your-org-id"         # Required
  token_type       = "Bearer"              # Options: Bearer (default), AxonApi
}
```

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `api_key` | string | No* | - | API key for authentication (*required for SaaS) |
| `axonops_host` | string | No | dash.axonops.cloud/\<org_id\> | AxonOps server hostname |
| `axonops_protocol` | string | No | https | Protocol (http/https) |
| `org_id` | string | Yes | - | Organization ID |
| `token_type` | string | No | Bearer | Authorization header type |

## Resources

### axonops_kafka_topic

Manages Kafka topics.

```hcl
resource "axonops_kafka_topic" "example" {
  name               = "my-topic"
  partitions         = 3
  replication_factor = 2
  cluster_name       = "my-kafka-cluster"
  config = {
    cleanup_policy      = "delete"
    retention_ms        = "604800000"
    delete_retention_ms = "86400000"
  }
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Topic name |
| `partitions` | int | Yes | Number of partitions (cannot be changed after creation) |
| `replication_factor` | int | Yes | Replication factor (cannot be changed after creation) |
| `cluster_name` | string | Yes | Kafka cluster name |
| `config` | map | No | Topic configurations (use underscores, converted to dots) |

### axonops_acl

Manages Kafka ACLs.

```hcl
resource "axonops_kafka_acl" "example" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = "my-topic"
  resource_pattern_type = "LITERAL"
  principal             = "User:alice"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}
```

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `cluster_name` | string | Yes | - | Kafka cluster name |
| `resource_type` | string | Yes | - | ANY, TOPIC, GROUP, CLUSTER, TRANSACTIONAL_ID, DELEGATION_TOKEN, USER |
| `resource_name` | string | Yes | - | Name of the resource |
| `resource_pattern_type` | string | No | LITERAL | ANY, MATCH, LITERAL, PREFIXED |
| `principal` | string | Yes | - | Principal (e.g., User:alice) |
| `host` | string | No | * | Host pattern |
| `operation` | string | Yes | - | READ, WRITE, CREATE, DELETE, ALTER, DESCRIBE, etc. |
| `permission_type` | string | Yes | - | ANY, DENY, ALLOW |

### axonops_connector

Manages Kafka Connect connectors.

```hcl
resource "axonops_kafka_connect_connector" "example" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "my-connector"
  config = {
    "connector.class" = "org.apache.kafka.connect.file.FileStreamSourceConnector"
    "tasks.max"       = "1"
    "file"            = "/tmp/input.txt"
    "topic"           = "my-topic"
  }
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster_name` | string | Yes | Kafka cluster name |
| `connect_cluster_name` | string | Yes | Kafka Connect cluster name |
| `name` | string | Yes | Connector name |
| `config` | map | Yes | Connector configuration |
| `type` | string | Computed | Connector type (source/sink) |

### axonops_slack_integration

Manages a Slack alerting integration. AxonOps delivers alerts to the configured Slack channel via an incoming webhook.

```hcl
resource "axonops_slack_integration" "ops_alerts" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-slack-alerts"
  webhook_url  = var.slack_webhook_url
  channel      = "#ops-alerts"
}
```

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `cluster_name` | string | Yes | - | Name of the cluster |
| `cluster_type` | string | Yes | - | Cluster type: `cassandra`, `kafka`, or `dse` |
| `name` | string | Yes | - | Unique name for this integration |
| `webhook_url` | string (sensitive) | Yes | - | Slack incoming webhook URL |
| `channel` | string | No | `""` | Slack channel name (e.g. `#ops-alerts`). When empty, the channel on the webhook is used |
| `axonops_url` | string | No | `""` | AxonOps dashboard URL override included in alert messages |
| `id` | string | Computed | - | Integration ID assigned by AxonOps |

### axonops_teams_integration

Manages a Microsoft Teams alerting integration. AxonOps delivers alerts to the configured Teams channel via an incoming webhook.

```hcl
resource "axonops_teams_integration" "ops_alerts" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-teams-alerts"
  webhook_url  = var.teams_webhook_url
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster_name` | string | Yes | Name of the cluster |
| `cluster_type` | string | Yes | Cluster type: `cassandra`, `kafka`, or `dse` |
| `name` | string | Yes | Unique name for this integration |
| `webhook_url` | string (sensitive) | Yes | Microsoft Teams incoming webhook URL |
| `id` | string | Computed | Integration ID assigned by AxonOps |

### axonops_pagerduty_integration

Manages a PagerDuty alerting integration. AxonOps creates PagerDuty incidents via the Events API v2 when alerts fire.

```hcl
resource "axonops_pagerduty_integration" "oncall" {
  cluster_name    = "production-kafka"
  cluster_type    = "kafka"
  name            = "pagerduty-oncall"
  integration_key = var.pagerduty_integration_key
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster_name` | string | Yes | Name of the cluster |
| `cluster_type` | string | Yes | Cluster type: `cassandra`, `kafka`, or `dse` |
| `name` | string | Yes | Unique name for this integration |
| `integration_key` | string (sensitive) | Yes | PagerDuty Events API v2 integration key |
| `id` | string | Computed | Integration ID assigned by AxonOps |

### axonops_opsgenie_integration

Manages an OpsGenie alerting integration. AxonOps creates OpsGenie alerts using the configured API key.

```hcl
resource "axonops_opsgenie_integration" "oncall" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "opsgenie-oncall"
  opsgenie_key = var.opsgenie_api_key
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster_name` | string | Yes | Name of the cluster |
| `cluster_type` | string | Yes | Cluster type: `cassandra`, `kafka`, or `dse` |
| `name` | string | Yes | Unique name for this integration |
| `opsgenie_key` | string (sensitive) | Yes | OpsGenie API key |
| `id` | string | Computed | Integration ID assigned by AxonOps |

### axonops_servicenow_integration

Manages a ServiceNow alerting integration. AxonOps creates ServiceNow incidents using the configured instance credentials.

> **Warning:** Store `password` in a secrets manager and reference it via a Terraform variable. Do not commit plaintext passwords in `.tf` files.

```hcl
resource "axonops_servicenow_integration" "incidents" {
  cluster_name  = "production-cassandra"
  cluster_type  = "cassandra"
  name          = "servicenow-incidents"
  instance_name = "mycompany"
  user          = "axonops-svc"
  password      = var.servicenow_password
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster_name` | string | Yes | Name of the cluster |
| `cluster_type` | string | Yes | Cluster type: `cassandra`, `kafka`, or `dse` |
| `name` | string | Yes | Unique name for this integration |
| `instance_name` | string | Yes | ServiceNow instance name (subdomain of `<instance>.service-now.com`) |
| `user` | string | Yes | ServiceNow username |
| `password` | string (sensitive) | Yes | ServiceNow password for the configured user |
| `id` | string | Computed | Integration ID assigned by AxonOps |

### axonops_schema

Manages Schema Registry schemas.

```hcl
resource "axonops_schema" "example" {
  cluster_name = "my-kafka-cluster"
  subject      = "my-topic-value"
  schema_type  = "AVRO"
  schema       = jsonencode({
    type      = "record"
    name      = "MyRecord"
    namespace = "com.example"
    fields    = [
      { name = "id", type = "int" },
      { name = "name", type = "string" }
    ]
  })
}
```

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `cluster_name` | string | Yes | Kafka cluster name |
| `subject` | string | Yes | Schema subject (e.g., topic-name-value) |
| `schema` | string | Yes | Schema definition |
| `schema_type` | string | Yes | AVRO, PROTOBUF, or JSON |
| `schema_id` | int | Computed | Schema ID from registry |
| `version` | int | Computed | Schema version number |

### axonops_cassandra_scheduled_repair

Manages Cassandra scheduled repair configuration. Updates are performed as delete-then-create since the API does not support in-place updates.

```hcl
resource "axonops_cassandra_scheduled_repair" "weekly" {
  cluster_name  = "my-cassandra-cluster"
  tag           = "weekly-repair"
  schedule_expr = "0 2 * * 0"
  parallelism   = "DC-Aware"
  incremental   = true
}
```

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `cluster_name` | string | Yes | - | Cassandra cluster name |
| `tag` | string | Yes | - | Unique tag to identify this repair |
| `schedule_expr` | string | Yes | - | Cron expression for the schedule |
| `keyspace` | string | No | `""` | Keyspace to repair (empty = all) |
| `tables` | list | No | `[]` | Tables to repair (empty = all) |
| `blacklisted_tables` | list | No | `[]` | Tables to exclude |
| `nodes` | list | No | `[]` | Specific nodes to repair |
| `segments_per_node` | int | No | `1` | Segments per node |
| `segmented` | bool | No | `false` | Use segmented repair |
| `incremental` | bool | No | `false` | Use incremental repair |
| `job_threads` | int | No | `1` | Number of job threads |
| `primary_range` | bool | No | `false` | Use primary range repair |
| `parallelism` | string | No | `Parallel` | Parallel, Sequential, or DC-Aware |
| `optimise_streams` | bool | No | `false` | Optimise repair streams |
| `specific_data_centers` | list | No | `[]` | Specific data centers to repair |
| `skip_paxos` | bool | No | `false` | Skip Paxos repair |
| `paxos_only` | bool | No | `false` | Only run Paxos repair |
| `repair_id` | string | Computed | - | Repair ID assigned by AxonOps |

## Example Usage

```hcl
terraform {
  required_providers {
    axonops = {
      source = "axonops/axonops"
    }
  }
}

provider "axonops" {
  api_key  = var.axonops_api_key
  org_id   = "my-organization"
  # axonops_host defaults to dash.axonops.cloud/<org_id>
  # token_type defaults to Bearer
}

# Create a topic
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

# Create an ACL for the topic
resource "axonops_kafka_acl" "events_read" {
  cluster_name          = "production-kafka"
  resource_type         = "TOPIC"
  resource_name         = axonops_kafka_topic.events.name
  resource_pattern_type = "LITERAL"
  principal             = "User:consumer-app"
  operation             = "READ"
  permission_type       = "ALLOW"
}

# Register a schema for the topic
resource "axonops_schema" "events_value" {
  cluster_name = "production-kafka"
  subject      = "${axonops_kafka_topic.events.name}-value"
  schema_type  = "AVRO"
  schema       = jsonencode({
    type      = "record"
    name      = "UserEvent"
    namespace = "com.example.events"
    fields    = [
      { name = "user_id", type = "string" },
      { name = "event_type", type = "string" },
      { name = "timestamp", type = "long" }
    ]
  })
}
```

## Importing Existing Resources

All resources support importing existing configurations into Terraform state.

### Import ID Formats

| Resource | Import ID Format |
|----------|------------------|
| `axonops_kafka_topic` | `cluster_name/topic_name` |
| `axonops_kafka_acl` | `cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type` |
| `axonops_kafka_connect_connector` | `cluster_name/connect_cluster_name/connector_name` |
| `axonops_schema` | `cluster_name/subject` |
| `axonops_logcollector` | `cluster_name/log_collector_name` |
| `axonops_healthcheck_tcp` | `cluster_name/healthcheck_name` |
| `axonops_healthcheck_http` | `cluster_name/healthcheck_name` |
| `axonops_healthcheck_shell` | `cluster_name/healthcheck_name` |
| `axonops_slack_integration` | `cluster_type/cluster_name/name` |
| `axonops_teams_integration` | `cluster_type/cluster_name/name` |
| `axonops_pagerduty_integration` | `cluster_type/cluster_name/name` |
| `axonops_opsgenie_integration` | `cluster_type/cluster_name/name` |
| `axonops_servicenow_integration` | `cluster_type/cluster_name/name` |
| `axonops_cassandra_scheduled_repair` | `cluster_name/tag` |

### Import Examples

```bash
# Import a topic
terraform import axonops_kafka_topic.my_topic "my-cluster/my-topic"

# Import an ACL
terraform import axonops_kafka_acl.my_acl "my-cluster/TOPIC/my-topic/LITERAL/User:alice/*/READ/ALLOW"

# Import a connector
terraform import axonops_kafka_connect_connector.my_connector "my-cluster/my-connect-cluster/my-connector"

# Import a schema
terraform import axonops_schema.my_schema "my-cluster/my-topic-value"

# Import a log collector
terraform import axonops_logcollector.my_logs "my-cluster/My Log Collector"

# Import healthchecks
terraform import axonops_healthcheck_tcp.my_check "my-cluster/My TCP Check"
terraform import axonops_healthcheck_http.my_http "my-cluster/My HTTP Check"
terraform import axonops_healthcheck_shell.my_shell "my-cluster/My Shell Check"

# Import integrations (format: cluster_type/cluster_name/name)
terraform import axonops_slack_integration.my_slack "cassandra/my-cluster/ops-slack-alerts"
terraform import axonops_teams_integration.my_teams "cassandra/my-cluster/ops-teams-alerts"
terraform import axonops_pagerduty_integration.my_pagerduty "kafka/my-cluster/pagerduty-oncall"
terraform import axonops_opsgenie_integration.my_opsgenie "cassandra/my-cluster/opsgenie-oncall"
terraform import axonops_servicenow_integration.my_servicenow "cassandra/my-cluster/servicenow-incidents"

# Import a scheduled repair
terraform import axonops_cassandra_scheduled_repair.my_repair "my-cluster/weekly-repair"
```

### Bulk Import Script

For importing an entire cluster, use the provided import script:

```bash
# Usage
./scripts/import-cluster.sh <axonops_host> <org_id> <cluster_name> <api_key> [output_dir]

# Example
./scripts/import-cluster.sh axonops.example.com:8080 myorg mycluster abc123 ./imported

# The script will:
# 1. Generate .tf files for all resources (topics, ACLs, log collectors, healthchecks)
# 2. Create an import_commands.sh script with all terraform import commands
# 3. Generate a provider.tf with your configuration
```

After running the script:
1. Review the generated `.tf` files in the output directory
2. Set your API key: `export TF_VAR_axonops_api_key='your-api-key'`
3. Initialize Terraform: `terraform init`
4. Run the import commands: `bash import_commands.sh`
5. Verify the state: `terraform plan` (should show no changes)

## Development

### Building

```bash
make build
```

### Testing

```bash
# Configure main.tf with your settings
terraform init
terraform plan
terraform apply
```

## License

Apache License 2.0

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

***

*This project may contain trademarks or logos for projects, products, or services. Any use of third-party trademarks or logos are subject to those third-party's policies. AxonOps is a registered trademark of AxonOps Limited. Apache, Apache Cassandra, Cassandra, Apache Spark, Spark, Apache TinkerPop, TinkerPop, Apache Kafka and Kafka are either registered trademarks or trademarks of the Apache Software Foundation or its subsidiaries in Canada, the United States and/or other countries. Elasticsearch is a trademark of Elasticsearch B.V., registered in the U.S. and in other countries. Docker is a trademark or registered trademark of Docker, Inc. in the United States and/or other countries.*
