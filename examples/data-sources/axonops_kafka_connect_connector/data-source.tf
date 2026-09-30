# Read an existing connector
data "axonops_kafka_connect_connector" "existing" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "my-connector"
}

# Output connector details
output "connector_type" {
  value = data.axonops_kafka_connect_connector.existing.type
}

output "connector_config" {
  value = data.axonops_kafka_connect_connector.existing.config
}

# Use connector data to verify configuration
output "connector_class" {
  value = data.axonops_kafka_connect_connector.existing.config["connector.class"]
}
