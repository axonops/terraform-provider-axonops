data "axonops_silence" "maintenance_window" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  id           = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}

output "silence_active" {
  value = data.axonops_silence.maintenance_window.active
}

output "silence_duration" {
  value = data.axonops_silence.maintenance_window.duration
}
