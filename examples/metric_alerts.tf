# Metric Alert Rule Examples

# ── Cassandra Examples ────────────────────────────────────────────────

# Alert on high CPU usage across the cluster (with full annotations and integrations)
resource "axonops_metric_alert_rule" "cassandra_high_cpu" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "High CPU Usage"
  dashboard      = "System"
  chart          = "Avg IO wait CPU per Host"
  operator       = ">="
  warning_value  = 75
  critical_value = 90
  duration       = "15m"

  annotations = {
    summary     = "CPU usage per host is >= than { limit } (current value: {{ $value }})"
    description = "CPU usage has exceeded threshold"
  }

  integrations = {
    type             = "pagerduty"
    routing          = ["team-ops"]
    override_info    = false
    override_warning = false
    override_error   = true
  }
}

# Alert on clients timing out.
resource "axonops_metric_alert_rule" "cassandra_client_read_timeout" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "High Read Latency"
  dashboard      = "Overview"
  chart          = "Client Read Timeouts Per Second"
  operator       = ">"
  warning_value  = 10
  critical_value = 50
  duration       = "5m"
  scope          = ["Read"]
  group_by       = ["dc", "host_id"]

  annotations = {
    description = "Client read timeouts occurring"
  }
}

# Alert on compaction pending tasks
resource "axonops_metric_alert_rule" "cassandra_compaction_pending" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "Compaction Pending Tasks"
  dashboard      = "Compactions"
  chart          = "Pending TP Compaction Tasks"
  operator       = ">"
  warning_value  = 50
  critical_value = 200
  duration       = "30m"
  group_by       = ["host_id"]

  annotations = {
    description = "Too many pending compaction tasks"
  }
}

# Alert on heap memory usage in a specific datacenter
resource "axonops_metric_alert_rule" "cassandra_heap_usage" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "High Heap Memory Usage"
  dashboard      = "System Resource Usage"
  chart          = "JVM Heap Utilization"
  operator       = ">="
  warning_value  = 75
  critical_value = 90
  duration       = "10m"
  dc             = ["dc1"]
  group_by       = ["host_id"]
  scope          = ["used"]
  annotations = {
    description = "JVM heap memory usage is too high"
  }
}


# ── Kafka Examples ────────────────────────────────────────────────────

# Alert on under-replicated partitions
resource "axonops_metric_alert_rule" "kafka_under_replicated" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "Under Replicated Partitions"
  dashboard      = "Kafka Replication"
  chart          = "Under Replicated Partitions"
  operator       = ">"
  warning_value  = 0
  critical_value = 5
  duration       = "10m"
  group_by       = ["host_id"]

  annotations = {
    description = "Kafka has under-replicated partitions"
  }
}

# Alert on high consumer lag for a specific topic
resource "axonops_metric_alert_rule" "kafka_consumer_lag" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "High Consumer Lag"
  dashboard      = "Consumer Groups"
  chart          = "Consumer Groups"
  operator       = ">"
  warning_value  = 10000
  critical_value = 100000
  duration       = "15m"
  topic          = ["testtopic"]
  annotations = {
    summary     = "Consumer lag is > than { limit } (current value: {{ $value }})"
    description = "Consumer group lag is too high"
  }

  integrations = {
    type             = ""
    routing          = []
    override_info    = false
    override_warning = false
    override_error   = false
  }
}

# Alert on high request latency
resource "axonops_metric_alert_rule" "kafka_request_latency" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "High Produce Request Latency"
  dashboard      = "Kafka Requests"
  chart          = "Produce Time"
  operator       = ">"
  warning_value  = 100
  critical_value = 500
  duration       = "10m"
  group_by       = ["host_id"]
  request        = ["Produce"]
  annotations = {
    description = "Kafka produce request latency is too high"
  }
}

# Alert on disk usage
resource "axonops_metric_alert_rule" "kafka_disk_usage" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "High Disk Usage"
  dashboard      = "System"
  chart          = "Disk % Usage $mountpoint"
  operator       = ">="
  warning_value  = 75
  critical_value = 90
  duration       = "15m"
  group_by       = ["host_id"]

  annotations = {
    description = "Broker disk usage is approaching capacity"
  }
}

# Alert filtered to a specific datacenter and rack
resource "axonops_metric_alert_rule" "kafka_cpu_dc1_rack1" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "CPU Usage - DC1 Rack1"
  dashboard      = "System Resource Usage"
  chart          = "CPU Usage"
  operator       = ">="
  warning_value  = 80
  critical_value = 95
  duration       = "10m"
  dc             = ["dc1"]
  rack           = ["rack1"]

  annotations = {
    description = "CPU usage in dc1/rack1 is too high"
  }
}

# ── Data Source ───────────────────────────────────────────────────────

# Read an existing alert rule (returns dashboard/chart names, correlation_id, etc.)
data "axonops_metric_alert_rule" "existing" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  id           = "some-alert-rule-id"
}
