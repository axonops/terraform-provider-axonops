# Cassandra Alerting Integrations
# Defines all alerting integration endpoints for a Cassandra cluster.
# Sensitive values (webhook URLs, API keys, passwords) must be supplied via
# variables or a secrets manager — never hardcoded.
# Ported from the axonops-ansible-collection Cassandra alert pack.


# ── Slack ─────────────────────────────────────────────────────────────

resource "axonops_slack_integration" "developer" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "slack-developer"
  webhook_url  = var.slack_webhook_url
  channel      = "#cassandra-developer"
}

resource "axonops_slack_integration" "ops" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "slack-ops"
  webhook_url  = var.slack_webhook_url
  channel      = "#cassandra-ops"
}

resource "axonops_slack_integration" "backups" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "slack-backups"
  webhook_url  = var.slack_webhook_url
  channel      = "#cassandra-backups"
}

resource "axonops_slack_integration" "metrics" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "slack-metrics"
  webhook_url  = var.slack_webhook_url
  channel      = "#cassandra-metrics"
}

resource "axonops_slack_integration" "repair" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "slack-repair"
  webhook_url  = var.slack_webhook_url
  channel      = "#cassandra-repair"
}


# ── PagerDuty ─────────────────────────────────────────────────────────

resource "axonops_pagerduty_integration" "developer" {
  cluster_name    = var.cluster_name
  cluster_type    = "cassandra"
  name            = "pagerduty-developer"
  integration_key = var.pagerduty_integration_key
}

resource "axonops_pagerduty_integration" "ops" {
  cluster_name    = var.cluster_name
  cluster_type    = "cassandra"
  name            = "pagerduty-ops"
  integration_key = var.pagerduty_integration_key
}


# ── OpsGenie ──────────────────────────────────────────────────────────

resource "axonops_opsgenie_integration" "developer" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "opsgenie-developer"
  opsgenie_key = var.opsgenie_api_key
}

resource "axonops_opsgenie_integration" "ops" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "opsgenie-ops"
  opsgenie_key = var.opsgenie_api_key
}


# ── ServiceNow ────────────────────────────────────────────────────────

resource "axonops_servicenow_integration" "developer" {
  cluster_name  = var.cluster_name
  cluster_type  = "cassandra"
  name          = "servicenow-developer"
  instance_name = "mycompany"
  user          = "axonops-dev"
  password      = var.servicenow_password
}

resource "axonops_servicenow_integration" "ops" {
  cluster_name  = var.cluster_name
  cluster_type  = "cassandra"
  name          = "servicenow-ops"
  instance_name = "mycompany"
  user          = "axonops-svc"
  password      = var.servicenow_password
}


# ── Microsoft Teams ───────────────────────────────────────────────────

resource "axonops_teams_integration" "developer" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "teams-developer"
  webhook_url  = var.teams_webhook_url
}

resource "axonops_teams_integration" "ops" {
  cluster_name = var.cluster_name
  cluster_type = "cassandra"
  name         = "teams-ops"
  webhook_url  = var.teams_webhook_url
}
