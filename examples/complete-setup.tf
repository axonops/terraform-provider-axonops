# Complete Setup Example
# This file demonstrates a complete Kafka setup with topics, ACLs, schemas, and monitoring

# Variables for reuse
locals {
  cluster_name         = "production-kafka"
  connect_cluster_name = "production-connect"
}

# =============================================================================
# Topics
# =============================================================================

# User events topic
resource "axonops_topic_resource" "user_events" {
  name               = "user-events"
  partitions         = 12
  replication_factor = 3
  cluster_name       = local.cluster_name

  config = {
    cleanup_policy      = "delete"
    retention_ms        = "604800000"  # 7 days
    min_insync_replicas = "2"
  }
}

# Order events topic
resource "axonops_topic_resource" "orders" {
  name               = "orders"
  partitions         = 6
  replication_factor = 3
  cluster_name       = local.cluster_name

  config = {
    cleanup_policy      = "delete"
    retention_ms        = "2592000000"  # 30 days
    min_insync_replicas = "2"
  }
}

# User profiles compacted topic
resource "axonops_topic_resource" "user_profiles" {
  name               = "user-profiles"
  partitions         = 6
  replication_factor = 3
  cluster_name       = local.cluster_name

  config = {
    cleanup_policy            = "compact"
    min_cleanable_dirty_ratio = "0.1"
  }
}

# Dead letter queue
resource "axonops_topic_resource" "dlq" {
  name               = "dead-letter-queue"
  partitions         = 3
  replication_factor = 3
  cluster_name       = local.cluster_name

  config = {
    cleanup_policy = "delete"
    retention_ms   = "2592000000"  # 30 days
  }
}

# =============================================================================
# Schemas
# =============================================================================

# User event schema
resource "axonops_schema" "user_events" {
  cluster_name = local.cluster_name
  subject      = "user-events-value"
  schema_type  = "AVRO"
  schema       = jsonencode({
    type      = "record"
    name      = "UserEvent"
    namespace = "com.example.events"
    fields    = [
      { name = "user_id", type = "string" },
      { name = "event_type", type = "string" },
      { name = "timestamp", type = "long" },
      { name = "payload", type = ["null", "string"], default = null }
    ]
  })
}

# Order schema
resource "axonops_schema" "orders" {
  cluster_name = local.cluster_name
  subject      = "orders-value"
  schema_type  = "AVRO"
  schema       = jsonencode({
    type      = "record"
    name      = "Order"
    namespace = "com.example.orders"
    fields    = [
      { name = "order_id", type = "string" },
      { name = "customer_id", type = "string" },
      { name = "total_amount", type = "double" },
      { name = "status", type = "string" },
      { name = "created_at", type = "long" }
    ]
  })
}

# =============================================================================
# ACLs
# =============================================================================

# Producer service - write to user-events
resource "axonops_acl" "user_service_produce" {
  cluster_name          = local.cluster_name
  resource_type         = "TOPIC"
  resource_name         = axonops_topic_resource.user_events.name
  resource_pattern_type = "LITERAL"
  principal             = "User:user-service"
  host                  = "*"
  operation             = "WRITE"
  permission_type       = "ALLOW"
}

# Analytics service - read from user-events
resource "axonops_acl" "analytics_consume" {
  cluster_name          = local.cluster_name
  resource_type         = "TOPIC"
  resource_name         = axonops_topic_resource.user_events.name
  resource_pattern_type = "LITERAL"
  principal             = "User:analytics-service"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}

# Analytics service - consumer group access
resource "axonops_acl" "analytics_group" {
  cluster_name          = local.cluster_name
  resource_type         = "GROUP"
  resource_name         = "analytics-consumer-group"
  resource_pattern_type = "LITERAL"
  principal             = "User:analytics-service"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}

# Order service - produce to orders topic
resource "axonops_acl" "order_service_produce" {
  cluster_name          = local.cluster_name
  resource_type         = "TOPIC"
  resource_name         = axonops_topic_resource.orders.name
  resource_pattern_type = "LITERAL"
  principal             = "User:order-service"
  host                  = "*"
  operation             = "WRITE"
  permission_type       = "ALLOW"
}

# =============================================================================
# Connectors
# =============================================================================

# S3 sink for archiving events
resource "axonops_connector" "s3_archive" {
  cluster_name         = local.cluster_name
  connect_cluster_name = local.connect_cluster_name
  name                 = "s3-event-archive"

  config = {
    "connector.class" = "io.confluent.connect.s3.S3SinkConnector"
    "tasks.max"       = "3"
    "topics"          = "user-events,orders"
    "s3.bucket.name"  = "kafka-event-archive"
    "s3.region"       = "us-east-1"
    "flush.size"      = "10000"
    "storage.class"   = "io.confluent.connect.s3.storage.S3Storage"
    "format.class"    = "io.confluent.connect.s3.format.json.JsonFormat"
  }
}

# =============================================================================
# Log Collectors
# =============================================================================

# Kafka server logs
resource "axonops_logcollector" "server_logs" {
  cluster_name          = local.cluster_name
  name                  = "Kafka Server Logs"
  filename              = "{{index . \"comp_jvm_kafka.logs.dir\"}}/server.log"
  date_format           = "yyyy-MM-dd HH:mm:ss,SSS"
  error_alert_threshold = 5
  supported_agent_types = ["broker", "kraft-broker"]
}

# =============================================================================
# Healthchecks
# =============================================================================

# Broker TCP check
resource "axonops_healthcheck_tcp" "broker" {
  cluster_name          = local.cluster_name
  name                  = "Kafka Broker Health"
  tcp                   = "0.0.0.0:9092"
  interval              = "30s"
  timeout               = "10s"
  supported_agent_types = ["broker", "kraft-broker"]
}

# Schema Registry HTTP check
resource "axonops_healthcheck_http" "schema_registry" {
  cluster_name          = local.cluster_name
  name                  = "Schema Registry Health"
  url                   = "http://localhost:8081/subjects"
  method                = "GET"
  expected_status       = 200
  interval              = "1m"
  timeout               = "30s"
  supported_agent_types = ["schema-registry"]
}

# Disk space shell check
resource "axonops_healthcheck_shell" "disk_space" {
  cluster_name = local.cluster_name
  name         = "Disk Space Check"
  script       = "test $(df --output=pcent /var/kafka-logs | tail -1 | tr -d ' %') -lt 85"
  interval     = "5m"
  timeout      = "30s"
}
