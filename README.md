# AxonOps Terraform Provider for Kafka

A Terraform provider for managing Apache Kafka resources through the AxonOps platform. This provider enables Infrastructure as Code (IaC) management of Kafka topics, ACLs, connectors, and schemas.

## Features

- **Topics**: Create, update, and delete Kafka topics with custom configurations
- **ACLs**: Manage Kafka Access Control Lists for fine-grained permissions
- **Connectors**: Deploy and manage Kafka Connect connectors
- **Schemas**: Register and version schemas in Schema Registry (AVRO, Protobuf, JSON)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23 (for building from source)
- Access to an AxonOps instance with Kafka cluster(s)

## Installation

### Building from Source

```bash
git clone https://github.com/axonops/axonops-kafka-tf.git
cd axonops-kafka-tf
go build -o terraform-provider-axonops
```

### Development Override

For local development, add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "hashicorp/axonops" = "/path/to/axonops-kafka-tf"
  }
  direct {}
}
```

### Install to Local Plugin Directory

To install the provider to your local Terraform plugin cache, you can either download a pre-built release or build from source.

#### Option 1: Download from GitHub Releases

```bash
# Set variables for your platform
OS="linux"       # or "darwin" for macOS, "windows" for Windows
ARCH="amd64"     # or "arm64" for ARM-based systems
VERSION="1.0.0"

# Download the release
curl -LO "https://github.com/axonops/axonops-kafka-tf/releases/download/v${VERSION}/terraform-provider-axonops_v${VERSION}_${OS}_${ARCH}.zip"

# Create the plugin directory
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/hashicorp/axonops/${VERSION}/${OS}_${ARCH}/

# Unzip to the plugin directory
unzip "terraform-provider-axonops_v${VERSION}_${OS}_${ARCH}.zip" -d ~/.terraform.d/plugins/registry.terraform.io/hashicorp/axonops/${VERSION}/${OS}_${ARCH}/
```

#### Option 2: Build from Source

```bash
# Set variables for your platform
OS="linux"       # or "darwin" for macOS, "windows" for Windows
ARCH="amd64"     # or "arm64" for ARM-based systems
VERSION="1.0.0"

# Clone and build
git clone https://github.com/axonops/axonops-kafka-tf.git
cd axonops-kafka-tf
go build -o terraform-provider-axonops

# Create the plugin directory
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/hashicorp/axonops/${VERSION}/${OS}_${ARCH}/

# Copy the binary
cp terraform-provider-axonops ~/.terraform.d/plugins/registry.terraform.io/hashicorp/axonops/${VERSION}/${OS}_${ARCH}/
```

#### Configure Terraform

Reference the provider in your Terraform configuration:

```hcl
terraform {
  required_providers {
    axonops = {
      source  = "hashicorp/axonops"
      version = "1.0.0"
    }
  }
}
```

Run `terraform init` to initialize the provider.

## Provider Configuration

```hcl
provider "axonops" {
  api_key          = "your-api-key"        # Required for AxonOps SaaS
  axonops_host     = "axonops.example.com" # Default: axonops.dev.com
  axonops_protocol = "https"               # Default: https
  org_id           = "your-org-id"         # Required
  token_type       = "AxonApi"             # Options: AxonApi (default), Bearer
}
```

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `api_key` | string | No* | - | API key for authentication (*required for SaaS) |
| `axonops_host` | string | No | axonops.dev.com | AxonOps server hostname |
| `axonops_protocol` | string | No | https | Protocol (http/https) |
| `org_id` | string | Yes | - | Organization ID |
| `token_type` | string | No | AxonApi | Authorization header type |

## Resources

### axonops_topic_resource

Manages Kafka topics.

```hcl
resource "axonops_topic_resource" "example" {
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
resource "axonops_acl" "example" {
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
resource "axonops_connector" "example" {
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

## Example Usage

```hcl
terraform {
  required_providers {
    axonops = {
      source = "hashicorp/axonops"
    }
  }
}

provider "axonops" {
  api_key          = var.axonops_api_key
  axonops_protocol = "https"
  axonops_host     = "axonops.example.com"
  org_id           = "my-organization"
}

# Create a topic
resource "axonops_topic_resource" "events" {
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
resource "axonops_acl" "events_read" {
  cluster_name          = "production-kafka"
  resource_type         = "TOPIC"
  resource_name         = axonops_topic_resource.events.name
  resource_pattern_type = "LITERAL"
  principal             = "User:consumer-app"
  operation             = "READ"
  permission_type       = "ALLOW"
}

# Register a schema for the topic
resource "axonops_schema" "events_value" {
  cluster_name = "production-kafka"
  subject      = "${axonops_topic_resource.events.name}-value"
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
| `axonops_topic_resource` | `cluster_name/topic_name` |
| `axonops_acl` | `cluster_name/resource_type/resource_name/resource_pattern_type/principal/host/operation/permission_type` |
| `axonops_connector` | `cluster_name/connect_cluster_name/connector_name` |
| `axonops_schema` | `cluster_name/subject` |
| `axonops_logcollector` | `cluster_name/log_collector_name` |
| `axonops_healthcheck_tcp` | `cluster_name/healthcheck_name` |
| `axonops_healthcheck_http` | `cluster_name/healthcheck_name` |
| `axonops_healthcheck_shell` | `cluster_name/healthcheck_name` |

### Import Examples

```bash
# Import a topic
terraform import axonops_topic_resource.my_topic "my-cluster/my-topic"

# Import an ACL
terraform import axonops_acl.my_acl "my-cluster/TOPIC/my-topic/LITERAL/User:alice/*/READ/ALLOW"

# Import a connector
terraform import axonops_connector.my_connector "my-cluster/my-connect-cluster/my-connector"

# Import a schema
terraform import axonops_schema.my_schema "my-cluster/my-topic-value"

# Import a log collector
terraform import axonops_logcollector.my_logs "my-cluster/My Log Collector"

# Import healthchecks
terraform import axonops_healthcheck_tcp.my_check "my-cluster/My TCP Check"
terraform import axonops_healthcheck_http.my_http "my-cluster/My HTTP Check"
terraform import axonops_healthcheck_shell.my_shell "my-cluster/My Shell Check"
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
