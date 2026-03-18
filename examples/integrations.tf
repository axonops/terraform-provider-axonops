# Examples: AxonOps alerting integrations
#
# This file demonstrates configuring all supported alerting integrations and
# routing alerts to them. Sensitive values (webhook URLs, API keys, passwords)
# MUST be supplied via variables or a secrets manager — never hardcoded.
#
# All integrations use the same import format: cluster_type/cluster_name/name

variable "slack_webhook_url" {
  description = "Slack incoming webhook URL."
  type        = string
  sensitive   = true
}

variable "teams_webhook_url" {
  description = "Microsoft Teams incoming webhook URL."
  type        = string
  sensitive   = true
}

variable "pagerduty_integration_key" {
  description = "PagerDuty Events API v2 integration key."
  type        = string
  sensitive   = true
}

variable "opsgenie_api_key" {
  description = "OpsGenie API key."
  type        = string
  sensitive   = true
}

variable "servicenow_password" {
  description = "ServiceNow user password."
  type        = string
  sensitive   = true
}

# ---------------------------------------------------------------------------
# Slack integration
# ---------------------------------------------------------------------------
resource "axonops_slack_integration" "ops_alerts" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-slack-alerts"
  webhook_url  = var.slack_webhook_url
  channel      = "#ops-alerts"
}

# ---------------------------------------------------------------------------
# Microsoft Teams integration
# ---------------------------------------------------------------------------
resource "axonops_teams_integration" "ops_alerts" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-teams-alerts"
  webhook_url  = var.teams_webhook_url
}

# ---------------------------------------------------------------------------
# PagerDuty integration
# ---------------------------------------------------------------------------
resource "axonops_pagerduty_integration" "oncall" {
  cluster_name    = "production-cassandra"
  cluster_type    = "cassandra"
  name            = "pagerduty-oncall"
  integration_key = var.pagerduty_integration_key
}

# ---------------------------------------------------------------------------
# OpsGenie integration
# ---------------------------------------------------------------------------
resource "axonops_opsgenie_integration" "oncall" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "opsgenie-oncall"
  opsgenie_key = var.opsgenie_api_key
}

# ---------------------------------------------------------------------------
# ServiceNow integration
# ---------------------------------------------------------------------------
resource "axonops_servicenow_integration" "incidents" {
  cluster_name  = "production-cassandra"
  cluster_type  = "cassandra"
  name          = "servicenow-incidents"
  instance_name = "mycompany"
  user          = "axonops-svc"
  password      = var.servicenow_password
}

# ---------------------------------------------------------------------------
# Alert routes — connect integrations to alert categories
# ---------------------------------------------------------------------------

# Send all error-level alerts to Slack
resource "axonops_alert_route" "slack_global" {
  cluster_name     = axonops_slack_integration.ops_alerts.cluster_name
  cluster_type     = axonops_slack_integration.ops_alerts.cluster_type
  integration_name = axonops_slack_integration.ops_alerts.name
  integration_type = "slack"
  type             = "global"
  severity         = "error"
}

# Page on-call via PagerDuty for error-level alerts
resource "axonops_alert_route" "pagerduty_global" {
  cluster_name     = axonops_pagerduty_integration.oncall.cluster_name
  cluster_type     = axonops_pagerduty_integration.oncall.cluster_type
  integration_name = axonops_pagerduty_integration.oncall.name
  integration_type = "pagerduty"
  type             = "global"
  severity         = "error"
}

# ---------------------------------------------------------------------------
# Data sources — read integrations managed in another Terraform root module
# ---------------------------------------------------------------------------

data "axonops_slack_integration" "shared" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-slack-alerts"
}

data "axonops_teams_integration" "shared" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-teams-alerts"
}

data "axonops_pagerduty_integration" "shared" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "pagerduty-oncall"
}

data "axonops_opsgenie_integration" "shared" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "opsgenie-oncall"
}

data "axonops_servicenow_integration" "shared" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "servicenow-incidents"
}
