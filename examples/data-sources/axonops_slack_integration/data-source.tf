data "axonops_slack_integration" "existing" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-slack-alerts"
}

output "slack_integration_id" {
  value = data.axonops_slack_integration.existing.id
}

output "slack_channel" {
  value = data.axonops_slack_integration.existing.channel
}
