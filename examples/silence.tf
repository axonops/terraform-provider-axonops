# Silence Window Examples

# Basic one-time silence for 1 hour
resource "axonops_silence" "maintenance" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  duration     = "1h"
}

# Recurring silence every day at midnight for 30 minutes
resource "axonops_silence" "nightly_maintenance" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  duration     = "30m"
  is_recurring = true
  cron_expr    = "0 0 * * *"
}

# Recurring silence on the first day of each month for 3 hours
resource "axonops_silence" "monthly_maintenance" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  duration     = "3h"
  is_recurring = true
  cron_expr    = "0 0 1 * *"
}

# Silence for specific datacenters
resource "axonops_silence" "dc_maintenance" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  duration     = "2h"
  datacenters  = ["dc1", "dc2"]
}

# Kafka cluster silence
resource "axonops_silence" "kafka_maintenance" {
  cluster_name = "my-kafka-cluster"
  cluster_type = "kafka"
  duration     = "1h30m"
  is_recurring = true
  cron_expr    = "0 2 * * 0" # Every Sunday at 2 AM
}

# Read an existing silence
data "axonops_silence" "existing" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  id           = "existing-silence-uuid"
}
