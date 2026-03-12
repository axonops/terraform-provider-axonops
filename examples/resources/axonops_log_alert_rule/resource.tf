# Basic log alert - detect error messages
resource "axonops_log_alert_rule" "error_logs" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "Error Log Alert"
  content        = "Exception"
  level          = "error"
  operator       = ">="
  warning_value  = 5
  critical_value = 20
  duration       = "15m"
  description    = "Alert when error logs exceed threshold"
}

# Alert for specific log content with multiple levels
resource "axonops_log_alert_rule" "connection_errors" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "Connection Error Alert"
  content        = "Connection refused"
  level          = "error,warning"
  operator       = ">="
  warning_value  = 3
  critical_value = 10
  duration       = "10m"
  description    = "Alert for connection failures"
}

# Alert filtering by log type and source
resource "axonops_log_alert_rule" "gc_warnings" {
  cluster_name   = "my-cassandra-cluster"
  cluster_type   = "cassandra"
  name           = "GC Warning Alert"
  content        = "GC pause"
  level          = "warning"
  log_type       = "gc"
  source         = "/var/log/cassandra/gc.log"
  operator       = ">="
  warning_value  = 10
  critical_value = 50
  duration       = "30m"
  description    = "Alert when garbage collection pauses are excessive"
}

# Kafka log alert example
resource "axonops_log_alert_rule" "kafka_errors" {
  cluster_name   = "my-kafka-cluster"
  cluster_type   = "kafka"
  name           = "Kafka Error Alert"
  content        = "ERROR"
  level          = "error"
  operator       = ">="
  warning_value  = 5
  critical_value = 25
  duration       = "15m"
  description    = "Alert for Kafka error messages"
}
