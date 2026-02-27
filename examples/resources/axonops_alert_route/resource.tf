# Global route - send all error alerts to PagerDuty
resource "axonops_alert_route" "global_pagerduty" {
  cluster_name     = "my-kafka-cluster"
  cluster_type     = "kafka"
  integration_name = "ops-pagerduty"
  integration_type = "pagerduty"
  type             = "global"
  severity         = "error"
}

# Category-specific route - send metric warnings to Slack with override
resource "axonops_alert_route" "metrics_slack" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "metrics-slack"
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
}

# Route without override
resource "axonops_alert_route" "nodes_webhook" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  integration_name = "monitoring-webhook"
  integration_type = "webhook"
  type             = "nodes"
  severity         = "info"
  enable_override  = false
}
