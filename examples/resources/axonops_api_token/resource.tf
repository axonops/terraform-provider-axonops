# Read-only token for one Kafka cluster, used by a CI pipeline
resource "axonops_api_token" "ci_readonly" {
  name          = "ci-readonly"
  allowed_roles = ["my-org/kafka/my-kafka-cluster/readonly"]
  expires_at    = "2027-01-01T00:00:00Z"
}

# Admin token rotated every 90 days (requires the hashicorp/time provider)
resource "time_rotating" "admin_token" {
  rotation_days = 90
}

resource "axonops_api_token" "automation" {
  name          = "automation"
  allowed_roles = ["my-org/cassandra/admin", "my-org/kafka/admin"]

  rotation_triggers = {
    rotated_at = time_rotating.admin_token.id
  }

  # Create the new token before the old one is revoked.
  lifecycle {
    create_before_destroy = true
  }
}

output "automation_token" {
  value     = axonops_api_token.automation.secret
  sensitive = true
}
