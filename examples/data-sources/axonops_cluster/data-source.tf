# Look up a cluster by name
data "axonops_cluster" "prod" {
  name = "prod-cassandra"
}

output "prod_cluster" {
  value = {
    id           = data.axonops_cluster.prod.id
    type         = data.axonops_cluster.prod.type
    status       = data.axonops_cluster.prod.status
    nodes        = data.axonops_cluster.prod.node_count
    active_nodes = data.axonops_cluster.prod.active_node_count
    datacenters  = data.axonops_cluster.prod.datacenters
    versions     = data.axonops_cluster.prod.versions
  }
}

# Disambiguate when a Cassandra and a Kafka cluster share a name
data "axonops_cluster" "shared_kafka" {
  name = "shared"
  type = "kafka"
}

# Use the looked-up cluster as a dependency for other resources
resource "axonops_kafka_topic" "events" {
  cluster_name       = data.axonops_cluster.shared_kafka.name
  name               = "events"
  partitions         = 6
  replication_factor = 3
}
