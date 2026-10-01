# List application keyspaces (system keyspaces excluded)
data "axonops_cassandra_keyspaces" "app" {
  cluster_name = "prod-cassandra"
}

output "keyspaces" {
  value = data.axonops_cassandra_keyspaces.app.names
}

# Include system keyspaces, on a DSE cluster
data "axonops_cassandra_keyspaces" "dse_all" {
  cluster_name   = "prod-dse"
  cluster_type   = "dse"
  include_system = true
}

# Map each keyspace to its tables
output "tables_by_keyspace" {
  value = { for ks in data.axonops_cassandra_keyspaces.app.keyspaces : ks.name => ks.tables }
}
