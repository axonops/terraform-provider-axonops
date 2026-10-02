# List every topic in a Kafka cluster
data "axonops_kafka_topics" "all" {
  cluster_name = "my-kafka-cluster"
}

output "topic_names" {
  value = data.axonops_kafka_topics.all.names
}

# Topics whose name starts with "orders-"
locals {
  order_topics = [
    for t in data.axonops_kafka_topics.all.topics : t.name
    if startswith(t.name, "orders-")
  ]
}

# Grant a consumer read access to every order topic
resource "axonops_kafka_acl" "orders_read" {
  for_each = toset(local.order_topics)

  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = each.value
  resource_pattern_type = "LITERAL"
  principal             = "User:orders-consumer"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}
