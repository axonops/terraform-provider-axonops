# List every cluster in the organisation
data "axonops_clusters" "all" {}

output "all_clusters" {
  value = data.axonops_clusters.all.clusters
}

# Only Cassandra clusters
data "axonops_clusters" "cassandra" {
  type = "cassandra"
}

# Clusters currently raising error-level alerts
output "red_clusters" {
  value = [for c in data.axonops_clusters.all.clusters : c.id if c.status == "red"]
}

# Apply the same Slack integration to every Cassandra cluster
resource "axonops_slack_integration" "ops" {
  for_each = toset(data.axonops_clusters.cassandra.names)

  cluster_name = each.value
  cluster_type = "cassandra"
  name         = "ops-alerts"
  webhook_url  = var.slack_webhook_url
  channel      = "#ops-alerts"
}

variable "slack_webhook_url" {
  type      = string
  sensitive = true
}
