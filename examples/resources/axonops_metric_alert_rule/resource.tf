# Basic alert - high CPU usage across the cluster
resource "axonops_metric_alert_rule" "high_cpu" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "High CPU Usage"
  metric         = "cpu_usage_percent"
  operator       = ">="
  warning_value  = 75
  critical_value = 90
  duration       = "15m"
  description    = "CPU usage has exceeded threshold"
}

# Alert with filters - high read latency by datacenter and host
resource "axonops_metric_alert_rule" "read_latency" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "High Read Latency"
  metric         = "org_apache_cassandra_metrics_client_request_latency"
  operator       = ">"
  warning_value  = 50
  critical_value = 100
  duration       = "10m"
  description    = "Read latency is above acceptable thresholds (ms)"
  scope          = ["Read"]
  percentile     = ["95thPercentile"]
  group_by       = ["dc", "host_id"]
}

# Kafka alert - under-replicated partitions
resource "axonops_metric_alert_rule" "under_replicated" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "Under-Replicated Partitions"
  metric         = "kafka_server_replica_manager_under_replicated_partitions"
  operator       = ">"
  warning_value  = 0
  critical_value = 5
  duration       = "10m"
  description    = "Kafka has under-replicated partitions"
  group_by       = ["host_id"]
}
