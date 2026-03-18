# Cassandra Scheduled Repair Examples

# Basic scheduled repair - runs on the first day of each month at midnight
resource "axonops_cassandra_scheduled_repair" "monthly" {
  cluster_name  = "my-cassandra-cluster"
  tag           = "monthly-full-repair"
  schedule_expr = "0 0 1 * *"
}

# Scheduled repair for a specific keyspace - runs every Sunday at 02:00
resource "axonops_cassandra_scheduled_repair" "keyspace_repair" {
  cluster_name  = "my-cassandra-cluster"
  tag           = "users-keyspace-repair"
  keyspace      = "users"
  schedule_expr = "0 2 * * 0"
  parallelism   = "DC-Aware"
  incremental   = true
}

# Scheduled repair with excluded tables and specific data centers
resource "axonops_cassandra_scheduled_repair" "selective_repair" {
  cluster_name  = "my-cassandra-cluster"
  tag           = "selective-repair"
  keyspace      = "analytics"
  schedule_expr = "0 3 * * 6"

  blacklisted_tables     = ["large_events", "raw_logs"]
  specific_data_centers  = ["dc1", "dc2"]
  segments_per_node      = 4
  segmented              = true
  optimise_streams       = true
  job_threads            = 2
}

# Read an existing scheduled repair
data "axonops_cassandra_scheduled_repair" "existing" {
  cluster_name = "my-cassandra-cluster"
  tag          = "monthly-full-repair"
}
