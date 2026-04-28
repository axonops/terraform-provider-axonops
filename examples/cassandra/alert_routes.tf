# Cassandra Alert Routes
# Connects alerting integrations to alert categories and severities.
# Ported from the axonops-ansible-collection Cassandra alert pack.
# Depends on integrations defined in integrations.tf.


# ── Global Routes ─────────────────────────────────────────────────────
# Catch-all routes for alerts not matched by a category override.

resource "axonops_alert_route" "global_opsgenie_error" {
  cluster_name     = axonops_opsgenie_integration.ops.cluster_name
  cluster_type     = axonops_opsgenie_integration.ops.cluster_type
  integration_name = axonops_opsgenie_integration.ops.name
  integration_type = "opsgenie"
  type             = "global"
  severity         = "error"
  enable_override  = false
}

resource "axonops_alert_route" "global_opsgenie_warning" {
  cluster_name     = axonops_opsgenie_integration.ops.cluster_name
  cluster_type     = axonops_opsgenie_integration.ops.cluster_type
  integration_name = axonops_opsgenie_integration.ops.name
  integration_type = "opsgenie"
  type             = "global"
  severity         = "warning"
  enable_override  = false
}

resource "axonops_alert_route" "global_slack_info" {
  cluster_name     = axonops_slack_integration.ops.cluster_name
  cluster_type     = axonops_slack_integration.ops.cluster_type
  integration_name = axonops_slack_integration.ops.name
  integration_type = "slack"
  type             = "global"
  severity         = "info"
  enable_override  = false
}


# ── Backup Routes ─────────────────────────────────────────────────────

resource "axonops_alert_route" "backups_slack_error" {
  cluster_name     = axonops_slack_integration.backups.cluster_name
  cluster_type     = axonops_slack_integration.backups.cluster_type
  integration_name = axonops_slack_integration.backups.name
  integration_type = "slack"
  type             = "backups"
  severity         = "error"
  enable_override  = true
}

resource "axonops_alert_route" "backups_slack_warning" {
  cluster_name     = axonops_slack_integration.backups.cluster_name
  cluster_type     = axonops_slack_integration.backups.cluster_type
  integration_name = axonops_slack_integration.backups.name
  integration_type = "slack"
  type             = "backups"
  severity         = "warning"
  enable_override  = true
}

resource "axonops_alert_route" "backups_slack_info" {
  cluster_name     = axonops_slack_integration.backups.cluster_name
  cluster_type     = axonops_slack_integration.backups.cluster_type
  integration_name = axonops_slack_integration.backups.name
  integration_type = "slack"
  type             = "backups"
  severity         = "info"
  enable_override  = true
}


# ── Metrics Routes ────────────────────────────────────────────────────

resource "axonops_alert_route" "metrics_slack_error" {
  cluster_name     = axonops_slack_integration.metrics.cluster_name
  cluster_type     = axonops_slack_integration.metrics.cluster_type
  integration_name = axonops_slack_integration.metrics.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "error"
  enable_override  = true
}

resource "axonops_alert_route" "metrics_slack_warning" {
  cluster_name     = axonops_slack_integration.metrics.cluster_name
  cluster_type     = axonops_slack_integration.metrics.cluster_type
  integration_name = axonops_slack_integration.metrics.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
  enable_override  = true
}

resource "axonops_alert_route" "metrics_slack_info" {
  cluster_name     = axonops_slack_integration.metrics.cluster_name
  cluster_type     = axonops_slack_integration.metrics.cluster_type
  integration_name = axonops_slack_integration.metrics.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "info"
  enable_override  = true
}


# ── Repair Routes ─────────────────────────────────────────────────────

resource "axonops_alert_route" "repairs_slack_error" {
  cluster_name     = axonops_slack_integration.repair.cluster_name
  cluster_type     = axonops_slack_integration.repair.cluster_type
  integration_name = axonops_slack_integration.repair.name
  integration_type = "slack"
  type             = "repairs"
  severity         = "error"
  enable_override  = true
}

resource "axonops_alert_route" "repairs_slack_warning" {
  cluster_name     = axonops_slack_integration.repair.cluster_name
  cluster_type     = axonops_slack_integration.repair.cluster_type
  integration_name = axonops_slack_integration.repair.name
  integration_type = "slack"
  type             = "repairs"
  severity         = "warning"
  enable_override  = true
}

resource "axonops_alert_route" "repairs_slack_info" {
  cluster_name     = axonops_slack_integration.repair.cluster_name
  cluster_type     = axonops_slack_integration.repair.cluster_type
  integration_name = axonops_slack_integration.repair.name
  integration_type = "slack"
  type             = "repairs"
  severity         = "info"
  enable_override  = true
}
