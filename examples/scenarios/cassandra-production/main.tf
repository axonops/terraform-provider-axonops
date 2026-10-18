# Production Cassandra cluster: nightly backup to S3, weekly repair,
# a metric alert, and routing of errors to Slack.

terraform {
  required_providers {
    axonops = {
      source = "axonops/axonops"
    }
  }
}

variable "org_id" {
  description = "AxonOps organisation ID."
  type        = string
}

variable "api_key" {
  description = "AxonOps API key."
  type        = string
  sensitive   = true
}

variable "cluster_name" {
  description = "Name of the Cassandra cluster in AxonOps."
  type        = string
  default     = "production-cassandra"
}

variable "s3_remote_config" {
  description = "Remote storage settings as key=value pairs separated by newlines."
  type        = string
  sensitive   = true
}

variable "slack_webhook_url" {
  description = "Slack incoming webhook URL."
  type        = string
  sensitive   = true
}

provider "axonops" {
  org_id  = var.org_id
  api_key = var.api_key
}

resource "axonops_cassandra_backup" "nightly" {
  cluster_name     = var.cluster_name
  tag              = "nightly"
  datacenters      = ["dc1"]
  schedule         = true
  schedule_expr    = "0 1 * * *"
  local_retention  = "7d"
  remote           = true
  remote_type      = "s3"
  remote_path      = "example-bucket/cassandra-backups"
  remote_retention = "90d"
  remote_config    = var.s3_remote_config
}

resource "axonops_cassandra_scheduled_repair" "weekly" {
  cluster_name  = var.cluster_name
  tag           = "weekly-repair"
  schedule_expr = "0 2 * * 0"
  parallelism   = "DC-Aware"
}

resource "axonops_metric_alert_rule" "high_cpu" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "High CPU Usage"
  dashboard      = "Cassandra Overview"
  chart          = "CPU Usage"
  operator       = ">="
  warning_value  = 75
  critical_value = 90
  duration       = "15m"

  annotations = {
    description = "CPU usage has exceeded the configured threshold."
  }
}

resource "axonops_slack_integration" "ops" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "ops-slack"
  webhook_url  = var.slack_webhook_url
  channel      = "#ops-alerts"
}

resource "axonops_alert_route" "errors_to_slack" {
  cluster_name     = axonops_slack_integration.ops.cluster_name
  cluster_type     = axonops_slack_integration.ops.cluster_type
  integration_name = axonops_slack_integration.ops.name
  integration_type = "slack"
  type             = "global"
  severity         = "error"
}
