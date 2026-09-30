data "axonops_kafka_acl" "producer" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = "my-topic"
  resource_pattern_type = "LITERAL"
  principal             = "User:producer-app"
  host                  = "*"
  operation             = "WRITE"
  permission_type       = "ALLOW"
}
