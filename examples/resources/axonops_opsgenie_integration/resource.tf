resource "axonops_opsgenie_integration" "oncall" {
  cluster_name = "production-cassandra"
  cluster_type = "cassandra"
  name         = "opsgenie-oncall"
  opsgenie_key = var.opsgenie_api_key
}

# Route node-level alerts to OpsGenie
resource "axonops_alert_route" "opsgenie_nodes" {
  cluster_name     = axonops_opsgenie_integration.oncall.cluster_name
  cluster_type     = axonops_opsgenie_integration.oncall.cluster_type
  integration_name = axonops_opsgenie_integration.oncall.name
  integration_type = "opsgenie"
  type             = "nodes"
  severity         = "error"
}
