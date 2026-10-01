# List every connector in a Kafka Connect cluster
data "axonops_kafka_connectors" "all" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
}

# Only sink connectors
data "axonops_kafka_connectors" "sinks" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  type                 = "sink"
}

output "sink_connector_names" {
  value = data.axonops_kafka_connectors.sinks.names
}

# Connectors that are not running
output "unhealthy_connectors" {
  value = [
    for c in data.axonops_kafka_connectors.all.connectors : c.name
    if c.state != "RUNNING"
  ]
}
