# List every schema registry subject
data "axonops_schemas" "all" {
  cluster_name = "my-kafka-cluster"
}

output "subjects" {
  value = data.axonops_schemas.all.subjects
}

# Only value schemas of the "orders" topics
data "axonops_schemas" "orders_values" {
  cluster_name  = "my-kafka-cluster"
  subject_regex = "^orders-.*-value$"
}

# Read the latest schema of each matching subject
data "axonops_schema" "orders" {
  for_each = toset(data.axonops_schemas.orders_values.subjects)

  cluster_name = "my-kafka-cluster"
  subject      = each.value
}

# Include soft-deleted subjects
data "axonops_schemas" "with_deleted" {
  cluster_name    = "my-kafka-cluster"
  include_deleted = true
}
