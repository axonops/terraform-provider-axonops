# Read an existing shell healthcheck
data "axonops_healthcheck_shell" "existing" {
  cluster_name = "my-kafka-cluster"
  name         = "Disk Space Check"
}

# Output healthcheck details
output "healthcheck_script" {
  value = data.axonops_healthcheck_shell.existing.script
}

output "healthcheck_shell" {
  value = data.axonops_healthcheck_shell.existing.shell
}

output "healthcheck_interval" {
  value = data.axonops_healthcheck_shell.existing.interval
}

output "healthcheck_timeout" {
  value = data.axonops_healthcheck_shell.existing.timeout
}
