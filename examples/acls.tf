# Kafka ACL Examples

# Allow a user to write to a specific topic
resource "axonops_acl" "producer" {
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
resource "axonops_acl" "consumer" {
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
resource "axonops_acl" "consumer_group" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "GROUP"
  resource_name         = "my-consumer-group"
  resource_pattern_type = "LITERAL"
  principal             = "User:consumer-app"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}

# Allow a user to read from all topics with a prefix
resource "axonops_acl" "prefixed_read" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = "events-"
  resource_pattern_type = "PREFIXED"
  principal             = "User:analytics-app"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}

# Allow a service account to describe topics
resource "axonops_acl" "describe" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = "*"
  resource_pattern_type = "LITERAL"
  principal             = "User:monitoring-service"
  host                  = "*"
  operation             = "DESCRIBE"
  permission_type       = "ALLOW"
}

# Deny access from a specific host
resource "axonops_acl" "deny_host" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "TOPIC"
  resource_name         = "sensitive-data"
  resource_pattern_type = "LITERAL"
  principal             = "User:*"
  host                  = "192.168.1.100"
  operation             = "READ"
  permission_type       = "DENY"
}

# Admin access to cluster
resource "axonops_acl" "admin_cluster" {
  cluster_name          = "my-kafka-cluster"
  resource_type         = "CLUSTER"
  resource_name         = "kafka-cluster"
  resource_pattern_type = "LITERAL"
  principal             = "User:admin"
  host                  = "*"
  operation             = "ALL"
  permission_type       = "ALLOW"
}
