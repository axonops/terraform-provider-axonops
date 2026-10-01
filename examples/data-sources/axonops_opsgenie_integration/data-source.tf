data "axonops_opsgenie_integration" "existing" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "opsgenie-oncall"
}

output "opsgenie_integration_id" {
  value = data.axonops_opsgenie_integration.existing.id
}
