# Alert Route Examples

# ── Global Routes ─────────────────────────────────────────────────────

# Send all info-level alerts to a Slack channel
resource "axonops_alert_route" "global_slack_info" {
  cluster_name     = "my-kafka-cluster"
  cluster_type     = "kafka"
  integration_name = "ops-slack"
  integration_type = "slack"
  type             = "global"
  severity         = "info"
}

# Send all error-level alerts to PagerDuty
resource "axonops_alert_route" "global_pagerduty_error" {
  cluster_name     = "my-kafka-cluster"
  cluster_type     = "kafka"
  integration_name = "ops-pagerduty"
  integration_type = "pagerduty"
  type             = "global"
  severity         = "error"
}

# Send all warning-level alerts to email
resource "axonops_alert_route" "global_email_warning" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "team-email"
  integration_type = "email"
  type             = "global"
  severity         = "warning"
}

# ── Category-Specific Routes (with override) ─────────────────────────

# Route metric alerts to Slack with override enabled (default)
resource "axonops_alert_route" "metrics_slack" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "metrics-slack"
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
}

# Route backup alerts to PagerDuty
resource "axonops_alert_route" "backups_pagerduty" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "ops-pagerduty"
  integration_type = "pagerduty"
  type             = "backups"
  severity         = "error"
}

# Route service check alerts to Teams
resource "axonops_alert_route" "servicechecks_teams" {
  cluster_name     = "my-kafka-cluster"
  cluster_type     = "kafka"
  integration_name = "ops-teams"
  integration_type = "teams"
  type             = "servicechecks"
  severity         = "error"
}

# Route node alerts to webhook without override
resource "axonops_alert_route" "nodes_webhook" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "monitoring-webhook"
  integration_type = "webhook"
  type             = "nodes"
  severity         = "info"
  enable_override  = false
}

# Route repair alerts to OpsGenie
resource "axonops_alert_route" "repairs_opsgenie" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "ops-opsgenie"
  integration_type = "opsgenie"
  type             = "repairs"
  severity         = "warning"
}

# Route rolling restart alerts to ServiceNow
resource "axonops_alert_route" "rollingrestart_servicenow" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "itsm-servicenow"
  integration_type = "servicenow"
  type             = "rollingrestart"
  severity         = "info"
}

# Route command alerts to SMTP
resource "axonops_alert_route" "commands_smtp" {
  cluster_name     = "my-kafka-cluster"
  cluster_type     = "kafka"
  integration_name = "alerts-smtp"
  integration_type = "smtp"
  type             = "commands"
  severity         = "error"
}
