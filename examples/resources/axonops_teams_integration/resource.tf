variable "teams_webhook_url" {
  type      = string
  sensitive = true
}

resource "axonops_teams_integration" "ops_alerts" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-teams-alerts"
  webhook_url  = var.teams_webhook_url
}

# Route backup alerts to the Teams integration
resource "axonops_alert_route" "teams_backups" {
  cluster_name     = axonops_teams_integration.ops_alerts.cluster_name
  cluster_type     = axonops_teams_integration.ops_alerts.cluster_type
  integration_name = axonops_teams_integration.ops_alerts.name
  integration_type = "teams"
  type             = "backups"
  severity         = "warning"
}
