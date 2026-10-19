# Alert routing: warnings to Slack, errors to PagerDuty, for one Kafka cluster.

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
  description = "Name of the Kafka cluster in AxonOps."
  type        = string
  default     = "production-kafka"
}

variable "slack_webhook_url" {
  description = "Slack incoming webhook URL."
  type        = string
  sensitive   = true
}

variable "pagerduty_integration_key" {
  description = "PagerDuty Events API v2 integration key."
  type        = string
  sensitive   = true
}

provider "axonops" {
  org_id  = var.org_id
  api_key = var.api_key
}

resource "axonops_slack_integration" "warnings" {
  cluster_name = var.cluster_name
  cluster_type = "kafka"
  name         = "kafka-warnings"
  webhook_url  = var.slack_webhook_url
  channel      = "#kafka-alerts"
}

resource "axonops_pagerduty_integration" "oncall" {
  cluster_name    = var.cluster_name
  cluster_type    = "kafka"
  name            = "kafka-oncall"
  integration_key = var.pagerduty_integration_key
}

resource "axonops_alert_route" "warnings_to_slack" {
  cluster_name     = axonops_slack_integration.warnings.cluster_name
  cluster_type     = axonops_slack_integration.warnings.cluster_type
  integration_name = axonops_slack_integration.warnings.name
  integration_type = "slack"
  type             = "global"
  severity         = "warning"
}

resource "axonops_alert_route" "errors_to_pagerduty" {
  cluster_name     = axonops_pagerduty_integration.oncall.cluster_name
  cluster_type     = axonops_pagerduty_integration.oncall.cluster_type
  integration_name = axonops_pagerduty_integration.oncall.name
  integration_type = "pagerduty"
  type             = "global"
  severity         = "error"
}
