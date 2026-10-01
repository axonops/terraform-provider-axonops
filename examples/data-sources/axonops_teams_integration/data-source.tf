data "axonops_teams_integration" "existing" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "ops-teams-alerts"
}

output "teams_integration_id" {
  value = data.axonops_teams_integration.existing.id
}
