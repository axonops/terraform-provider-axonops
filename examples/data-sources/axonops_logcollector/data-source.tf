# Read an existing log collector
data "axonops_logcollector" "existing" {
  cluster_name = "my-kafka-cluster"
  cluster_type = "kafka"
  name         = "Kafka Server Log"
}

# Output log collector details
output "logcollector_filename" {
  value = data.axonops_logcollector.existing.filename
}

output "logcollector_date_format" {
  value = data.axonops_logcollector.existing.date_format
}

output "logcollector_agent_types" {
  value = data.axonops_logcollector.existing.supported_agent_types
}

output "logcollector_error_regex" {
  value = data.axonops_logcollector.existing.error_regex
}

output "logcollector_error_threshold" {
  value = data.axonops_logcollector.existing.error_alert_threshold
}
