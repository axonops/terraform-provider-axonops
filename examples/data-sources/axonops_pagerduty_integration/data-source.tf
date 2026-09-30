data "axonops_pagerduty_integration" "existing" {
  cluster_name = "production-kafka"
  cluster_type = "kafka"
  name         = "pagerduty-oncall"
}

output "pagerduty_integration_id" {
  value = data.axonops_pagerduty_integration.existing.id
}
