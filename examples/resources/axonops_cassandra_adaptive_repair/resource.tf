# Basic adaptive repair
resource "axonops_cassandra_adaptive_repair" "basic" {
  cluster_name = "my-cassandra-cluster"
  active       = true
}

# Custom configuration
resource "axonops_cassandra_adaptive_repair" "custom" {
  cluster_name       = "my-cassandra-cluster"
  active             = true
  parallelism        = 5
  gc_grace_threshold = 43200 # 12 hours
  segment_retries    = 5
}

# With table exclusions
resource "axonops_cassandra_adaptive_repair" "with_exclusions" {
  cluster_name       = "my-cassandra-cluster"
  active             = true
  parallelism        = 8
  filter_twcs_tables = true
  blacklisted_tables = [
    "my_keyspace.large_table",
    "analytics.raw_events",
  ]
}
