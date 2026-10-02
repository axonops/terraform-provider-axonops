# Global default for every subject of the cluster
resource "axonops_schema_registry_compatibility" "global" {
  cluster_name        = "my-kafka-cluster"
  compatibility_level = "BACKWARD"
}

# Stricter level for one subject
resource "axonops_schema_registry_compatibility" "orders" {
  cluster_name        = "my-kafka-cluster"
  subject             = axonops_schema.orders.subject
  compatibility_level = "FULL_TRANSITIVE"
}

resource "axonops_schema" "orders" {
  cluster_name = "my-kafka-cluster"
  subject      = "orders-value"
  schema_type  = "AVRO"
  schema = jsonencode({
    type   = "record"
    name   = "Order"
    fields = [{ name = "id", type = "string" }]
  })
}
