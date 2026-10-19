# Production Kafka cluster: a topic, ACLs for its producer and consumer,
# and an under-replicated partitions alert.

terraform {
  required_providers {
    axonops = {
      source = "axonops/axonops"
    }
  }
}

variable "org_id" {
  description = "AxonOps organisation ID."
  type        = string
}

variable "api_key" {
  description = "AxonOps API key."
  type        = string
  sensitive   = true
}

variable "cluster_name" {
  description = "Name of the Kafka cluster in AxonOps."
  type        = string
  default     = "production-kafka"
}

provider "axonops" {
  org_id  = var.org_id
  api_key = var.api_key
}

resource "axonops_kafka_topic" "orders" {
  cluster_name       = var.cluster_name
  name               = "orders"
  partitions         = 6
  replication_factor = 3

  config = {
    retention_ms        = "604800000"
    min_insync_replicas = "2"
  }
}

resource "axonops_kafka_acl" "orders_producer" {
  cluster_name          = var.cluster_name
  resource_type         = "TOPIC"
  resource_name         = axonops_kafka_topic.orders.name
  resource_pattern_type = "LITERAL"
  principal             = "User:orders-producer"
  host                  = "*"
  operation             = "WRITE"
  permission_type       = "ALLOW"
}

resource "axonops_kafka_acl" "orders_consumer" {
  cluster_name          = var.cluster_name
  resource_type         = "TOPIC"
  resource_name         = axonops_kafka_topic.orders.name
  resource_pattern_type = "LITERAL"
  principal             = "User:orders-consumer"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}

resource "axonops_kafka_acl" "orders_consumer_group" {
  cluster_name          = var.cluster_name
  resource_type         = "GROUP"
  resource_name         = "orders-consumers"
  resource_pattern_type = "LITERAL"
  principal             = "User:orders-consumer"
  host                  = "*"
  operation             = "READ"
  permission_type       = "ALLOW"
}

resource "axonops_metric_alert_rule" "under_replicated" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Under-Replicated Partitions"
  dashboard      = "Kafka Overview"
  chart          = "Under-Replicated Partitions"
  operator       = ">"
  warning_value  = 0
  critical_value = 5
  duration       = "10m"
  group_by       = ["host_id"]
}
