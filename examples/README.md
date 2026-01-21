# AxonOps Terraform Provider Examples

This directory contains example Terraform configurations for the AxonOps Kafka Terraform provider.

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
| [complete-setup.tf](complete-setup.tf) | Complete example combining all resource types |

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

### axonops_topic_resource

Manages Kafka topics with configurations like partitions, replication factor, and topic-level settings.

### axonops_acl

Manages Kafka Access Control Lists for authorization.

### axonops_connector

Manages Kafka Connect connectors (source and sink).

### axonops_schema

Manages schemas in Schema Registry (supports AVRO, JSON, and PROTOBUF).

### axonops_logcollector

Configures log collection for monitoring Kafka logs.

### axonops_healthcheck_tcp

TCP connectivity healthchecks.

### axonops_healthcheck_http

HTTP endpoint healthchecks.

### axonops_healthcheck_shell

Shell script healthchecks.

## Notes

- Replace placeholder values (e.g., `my-kafka-cluster`, `your-api-key-here`) with actual values
- Topic `partitions` and `replication_factor` cannot be changed after creation
- Config keys in topics use underscores in Terraform (converted to dots for Kafka API)
- Healthchecks and log collectors share the same API pattern where updates replace all entries
