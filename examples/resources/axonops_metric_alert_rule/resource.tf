# Basic alert - high CPU usage across the cluster
resource "axonops_metric_alert_rule" "high_cpu" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "High CPU Usage"
  dashboard      = "Cassandra Overview"
  chart          = "CPU Usage"
  operator       = ">="
  warning_value  = 75
  critical_value = 90
  duration       = "15m"

  annotations = {
    description = "CPU usage has exceeded the configured threshold."
  }
}

# Alert with filters - high read latency by datacenter and host
resource "axonops_metric_alert_rule" "read_latency" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "High Read Latency"
  dashboard      = "Cassandra Overview"
  chart          = "Client Request Latency"
  operator       = ">"
  warning_value  = 50
  critical_value = 100
  duration       = "10m"
  scope          = ["Read"]
  percentile     = ["95thPercentile"]
  group_by       = ["dc", "host_id"]

  annotations = {
    description = "Read latency is above acceptable thresholds (ms)."
  }
}

# Kafka alert - under-replicated partitions, routed to PagerDuty
resource "axonops_metric_alert_rule" "under_replicated" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "Under-Replicated Partitions"
  dashboard      = "Kafka Overview"
  chart          = "Under-Replicated Partitions"
  operator       = ">"
  warning_value  = 0
  critical_value = 5
  duration       = "10m"
  group_by       = ["host_id"]

  integrations = {
    type    = "pagerduty"
    routing = ["ops-pagerduty"]
  }
}
