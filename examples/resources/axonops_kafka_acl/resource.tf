# Allow a user to write to a specific topic
resource "axonops_kafka_acl" "producer" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = "my-topic"
  resource_pattern_type = "LITERAL"
  principal             = "User:producer-app"
  host                  = "*"
  operation             = "WRITE"
  permission_type       = "ALLOW"
}

# Allow a user to read from a specific topic
resource "axonops_kafka_acl" "consumer" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = "my-topic"
  resource_pattern_type = "LITERAL"
  principal             = "User:consumer-app"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}

# Allow consumer group access
resource "axonops_kafka_acl" "consumer_group" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "GROUP"
  resource_name         = "my-consumer-group"
  resource_pattern_type = "LITERAL"
  principal             = "User:consumer-app"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}
