# Use the AxonOps defaults (warning after 30s, error after 1m)
resource "axonops_cassandra_agent_disconnect_tolerance" "default" {
  cluster_name = "my-cassandra-cluster"
}

# Tolerate longer disconnects, e.g. on a cluster with a flaky network link
resource "axonops_cassandra_agent_disconnect_tolerance" "relaxed" {
  cluster_name  = "my-dse-cluster"
  cluster_type  = "dse"
  warn_timeout  = "2m"
  error_timeout = "10m"
}
