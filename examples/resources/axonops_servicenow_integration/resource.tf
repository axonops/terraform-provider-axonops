resource "axonops_servicenow_integration" "incidents" {
  cluster_name  = "production-cassandra"
  cluster_type  = "cassandra"
  name          = "servicenow-incidents"
  instance_name = "mycompany"
  user          = "axonops-svc"
  password      = var.servicenow_password
}

# Route all error-level alerts to ServiceNow
resource "axonops_alert_route" "servicenow_global" {
  cluster_name     = axonops_servicenow_integration.incidents.cluster_name
  cluster_type     = axonops_servicenow_integration.incidents.cluster_type
  integration_name = axonops_servicenow_integration.incidents.name
  integration_type = "servicenow"
  type             = "global"
  severity         = "error"
}
