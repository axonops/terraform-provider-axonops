resource "axonops_slack_integration" "ops_alerts" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-slack-alerts"
  webhook_url  = var.slack_webhook_url
  channel      = "#ops-alerts"
}

# Route all error-level alerts to the Slack integration
resource "axonops_alert_route" "slack_global" {
  cluster_name     = axonops_slack_integration.ops_alerts.cluster_name
  cluster_type     = axonops_slack_integration.ops_alerts.cluster_type
  integration_name = axonops_slack_integration.ops_alerts.name
  integration_type = "slack"
  type             = "global"
  severity         = "error"
}
