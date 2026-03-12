# Read an existing log alert rule by ID
data "axonops_log_alert_rule" "example" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  id           = "existing-alert-id"
}

# Output the alert rule details
output "alert_name" {
  value = data.axonops_log_alert_rule.example.name
}

output "alert_duration" {
  value = data.axonops_log_alert_rule.example.duration
}
