---
page_title: "Getting Started - Terraform AxonOps Provider"
description: |-
  A step-by-step guide to using the Terraform AxonOps provider to manage Kafka and Cassandra infrastructure.
---

# Getting Started with Terraform AxonOps Provider

This guide walks you through configuring the AxonOps provider and creating your first resources.

## Prerequisites

- AxonOps SaaS account or self-hosted deployment
- AxonOps API key (generate from the AxonOps dashboard)
- Your AxonOps organization ID
- Terraform 1.5 or later (for `import` blocks; earlier versions work with `terraform import`)
- Existing Kafka or Cassandra clusters registered in AxonOps

## Step 1: Configure the Provider

Create a `terraform.tf` file with the required provider configuration:

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

Create a `variables.tf` file to store your credentials securely:

```terraform
variable "axonops_org_id" {
  description = "AxonOps organization ID"
  type        = string
  sensitive   = true
}

variable "axonops_api_key" {
  description = "AxonOps API key"
  type        = string
  sensitive   = true
}
```

Create a `terraform.tfvars` file (and add it to `.gitignore`):

```hcl
axonops_org_id = "your-org-id"
axonops_api_key = "your-api-key"
```

Or use environment variables:

```bash
export TF_VAR_axonops_org_id="your-org-id"
export TF_VAR_axonops_api_key="your-api-key"
```

## Step 2: Create Your First Kafka Topic

Create a `kafka.tf` file. Use underscores instead of dots in topic config keys:

```terraform
resource "axonops_kafka_topic" "orders" {
  cluster_name       = "my-kafka-cluster"
  name               = "orders"
  partitions         = 3
  replication_factor = 3

  config = {
    retention_ms   = "604800000" # 7 days
    cleanup_policy = "delete"
  }
}
```

-> **Note:** You can increase `partitions` later without recreating the topic. Kafka cannot reduce partitions, so a decrease fails at plan time.

## Step 3: Create an Alert Integration

Before routing alerts, define where they go. This example uses Slack:

```terraform
variable "slack_webhook_url" {
  description = "Slack incoming webhook URL"
  type        = string
  sensitive   = true
}

resource "axonops_slack_integration" "ops" {
  cluster_name = "my-kafka-cluster"
  cluster_type = "kafka"
  name         = "ops-slack"
  webhook_url  = var.slack_webhook_url
  channel      = "#ops-alerts"
}
```

## Step 4: Create a Metric Alert Rule

Alert rules are attached to a chart on an AxonOps dashboard. `dashboard` and `chart` must match the names shown in the AxonOps UI:

```terraform
resource "axonops_metric_alert_rule" "under_replicated" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "Under-Replicated Partitions"
  dashboard      = "Kafka Overview"
  chart          = "Under-Replicated Partitions"
  operator       = ">"
  warning_value  = 0
  critical_value = 5
  duration       = "10m"

  annotations = {
    description = "Partitions are under-replicated; check broker health."
  }
}
```

## Step 5: Route Alerts to Slack

Send warning-level metric alerts for the cluster to the Slack integration:

```terraform
resource "axonops_alert_route" "metrics_to_slack" {
  cluster_name     = axonops_slack_integration.ops.cluster_name
  cluster_type     = axonops_slack_integration.ops.cluster_type
  integration_name = axonops_slack_integration.ops.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
}
```

## Step 6: Schedule a Cassandra Backup

For Cassandra clusters, schedule a daily snapshot kept locally for 10 days:

```terraform
resource "axonops_cassandra_backup" "daily" {
  cluster_name    = "my-cassandra-cluster"
  tag             = "daily-backup"
  datacenters     = ["dc1"]
  schedule        = true
  schedule_expr   = "0 1 * * *" # daily at 01:00
  local_retention = "10d"
}
```

## Step 7: Initialize and Deploy

Initialize your Terraform working directory:

```bash
terraform init
```

Review the planned changes:

```bash
terraform plan
```

Apply the configuration:

```bash
terraform apply
```

## Security Best Practices

1. **Never commit credentials** - Use `.tfvars` files marked in `.gitignore` or environment variables
2. **Use a remote backend** - Store state in Terraform Cloud, S3, or another remote backend
3. **Enable state encryption** - Ensure your backend encrypts state at rest
4. **Rotate API keys regularly** - Update your AxonOps API key periodically
5. **Use TLS in production** - Only set `tls_skip_verify = true` for non-production environments with self-signed certificates

## Next Steps

- Explore [alert rules](https://registry.terraform.io/providers/axonops/axonops/latest/docs/resources/metric_alert_rule) for comprehensive monitoring
- Learn about [Kafka ACLs](https://registry.terraform.io/providers/axonops/axonops/latest/docs/resources/kafka_acl) for fine-grained access control
- Set up [silences](https://registry.terraform.io/providers/axonops/axonops/latest/docs/resources/silence) for maintenance windows
- Review [all resources and data sources](https://registry.terraform.io/providers/axonops/axonops/latest/docs) in the documentation
