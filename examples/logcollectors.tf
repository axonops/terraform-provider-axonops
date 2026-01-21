# Log Collector Examples

# Kafka server log collector
resource "axonops_logcollector" "server_log" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka Server Log"
  filename              = "{{index . \"comp_jvm_kafka.logs.dir\"}}/server.log"
  date_format           = "yyyy-MM-dd HH:mm:ss,SSS"
  supported_agent_types = ["broker", "kraft-broker"]
}

# Kafka controller log collector
resource "axonops_logcollector" "controller_log" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka Controller Log"
  filename              = "{{index . \"comp_jvm_kafka.logs.dir\"}}/controller.log"
  date_format           = "yyyy-MM-dd HH:mm:ss,SSS"
  supported_agent_types = ["kraft-controller"]
}

# Schema Registry log collector
resource "axonops_logcollector" "schema_registry_log" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Schema Registry Log"
  filename              = "{{.comp_log_file}}/schema-registry.log"
  date_format           = "yyyy-MM-dd HH:mm:ss,SSS"
  supported_agent_types = ["schema-registry"]
}

# AxonOps agent log collector
resource "axonops_logcollector" "agent_log" {
  cluster_name          = "my-kafka-cluster"
  name                  = "AxonOps Agent Log"
  filename              = "/var/log/axonops/axon-agent.log"
  date_format           = "yyyy-MM-ddTHH:mm:ssZ"
  supported_agent_types = ["all"]
}

# Custom application log with regex patterns
resource "axonops_logcollector" "app_log" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Custom Application Log"
  filename              = "/var/log/myapp/application.log"
  date_format           = "yyyy-MM-dd HH:mm:ss,SSS"
  info_regex            = "\\[INFO\\]"
  warning_regex         = "\\[WARN\\]"
  error_regex           = "\\[ERROR\\]"
  debug_regex           = "\\[DEBUG\\]"
  error_alert_threshold = 10
  supported_agent_types = ["broker", "kraft-broker"]
}

# Kafka Connect log collector
resource "axonops_logcollector" "connect_log" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka Connect Log"
  filename              = "/var/log/kafka-connect/connect.log"
  date_format           = "yyyy-MM-dd HH:mm:ss,SSS"
  supported_agent_types = ["all"]
}

# GC log collector
resource "axonops_logcollector" "gc_log" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka GC Log"
  filename              = "{{index . \"comp_jvm_kafka.logs.dir\"}}/kafka-gc.log"
  date_format           = "yyyy-MM-ddTHH:mm:ss"
  supported_agent_types = ["broker", "kraft-broker", "kraft-controller"]
}
