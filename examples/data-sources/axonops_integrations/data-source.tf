# List every integration configured for a cluster
data "axonops_integrations" "all" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
}

# Only Slack integrations
data "axonops_integrations" "slack" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  type         = "slack"
}

# Route error-level metric alerts to every Slack integration
resource "axonops_alert_route" "slack_errors" {
  for_each = toset([for i in data.axonops_integrations.slack.integrations : i.name])

  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = each.value
  integration_type = "slack"
  type             = "metrics"
  severity         = "error"
}
