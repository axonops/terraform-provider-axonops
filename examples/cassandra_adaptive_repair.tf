# Cassandra Adaptive Repair Examples

# Basic adaptive repair with defaults
resource "axonops_cassandra_adaptive_repair" "basic" {
  cluster_name = "my-cassandra-cluster"
  active       = true
}

# Adaptive repair with custom parallelism and thresholds
resource "axonops_cassandra_adaptive_repair" "custom" {
  cluster_name       = "my-cassandra-cluster"
  active             = true
  parallelism        = 5
  gc_grace_threshold = 43200  # 12 hours
  segment_retries    = 5
}

# Adaptive repair excluding specific tables
resource "axonops_cassandra_adaptive_repair" "with_exclusions" {
  cluster_name      = "my-cassandra-cluster"
  active            = true
  parallelism       = 8
  blacklisted_tables = [
    "my_keyspace.large_table",
    "analytics.raw_events",
    "logging.audit_log",
  ]
  filter_twcs_tables = true
}

# Fine-tuned repair with segment configuration
resource "axonops_cassandra_adaptive_repair" "fine_tuned" {
  cluster_name         = "my-cassandra-cluster"
  active               = true
  parallelism          = 15
  gc_grace_threshold   = 86400  # 24 hours
  segment_retries      = 3
  segments_per_vnode   = 2
  segment_target_size_mb = 512
  filter_twcs_tables   = true
  blacklisted_tables   = []
}

# DSE cluster adaptive repair
resource "axonops_cassandra_adaptive_repair" "dse" {
  cluster_name       = "my-dse-cluster"
  cluster_type       = "dse"
  active             = true
  parallelism        = 10
  gc_grace_threshold = 86400
}

# Conservative repair for production (low parallelism, small segments)
resource "axonops_cassandra_adaptive_repair" "production" {
  cluster_name           = "production-cassandra"
  active                 = true
  parallelism            = 3
  gc_grace_threshold     = 172800  # 48 hours
  segment_retries        = 5
  segments_per_vnode     = 1
  segment_target_size_mb = 128
  filter_twcs_tables     = true
  blacklisted_tables = [
    "system.large_partitions",
  ]
}

# Read existing adaptive repair settings
data "axonops_cassandra_adaptive_repair" "existing" {
  cluster_name = "my-cassandra-cluster"
}
