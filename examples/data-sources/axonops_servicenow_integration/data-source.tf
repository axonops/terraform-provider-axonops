data "axonops_servicenow_integration" "existing" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "servicenow-incidents"
}

output "servicenow_integration_id" {
  value = data.axonops_servicenow_integration.existing.id
}

output "servicenow_instance" {
  value = data.axonops_servicenow_integration.existing.instance_name
}
