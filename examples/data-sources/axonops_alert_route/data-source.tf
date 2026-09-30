data "axonops_alert_route" "metrics_pagerduty" {
  cluster_name     = "my-cassandra-cluster"
  cluster_type     = "cassandra"
  type             = "metrics"
  severity         = "warning"
  integration_type = "pagerduty"
  integration_name = "ops-pagerduty"
}

output "override_enabled" {
  value = data.axonops_alert_route.metrics_pagerduty.enable_override
}
