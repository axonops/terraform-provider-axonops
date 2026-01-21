# Schema Registry Examples

# Simple Avro schema for user events
resource "axonops_schema" "user_events" {
  cluster_name = "my-kafka-cluster"
  subject      = "user-events-value"
  schema_type  = "AVRO"
  schema       = jsonencode({
    type      = "record"
    name      = "UserEvent"
    namespace = "com.example.events"
    fields    = [
      {
        name = "user_id"
        type = "string"
      },
      {
        name = "event_type"
        type = "string"
      },
      {
        name = "timestamp"
        type = "long"
      },
      {
        name    = "metadata"
        type    = ["null", "string"]
        default = null
      }
    ]
  })
}

# Avro schema with complex types
resource "axonops_schema" "order" {
  cluster_name = "my-kafka-cluster"
  subject      = "orders-value"
  schema_type  = "AVRO"
  schema       = jsonencode({
    type      = "record"
    name      = "Order"
    namespace = "com.example.orders"
    fields    = [
      {
        name = "order_id"
        type = "string"
      },
      {
        name = "customer_id"
        type = "string"
      },
      {
        name = "items"
        type = {
          type  = "array"
          items = {
            type   = "record"
            name   = "OrderItem"
            fields = [
              { name = "product_id", type = "string" },
              { name = "quantity", type = "int" },
              { name = "price", type = "double" }
            ]
          }
        }
      },
      {
        name = "total_amount"
        type = "double"
      },
      {
        name = "status"
        type = {
          type    = "enum"
          name    = "OrderStatus"
          symbols = ["PENDING", "CONFIRMED", "SHIPPED", "DELIVERED", "CANCELLED"]
        }
      },
      {
        name = "created_at"
        type = {
          type        = "long"
          logicalType = "timestamp-millis"
        }
      }
    ]
  })
}

# JSON Schema example
resource "axonops_schema" "notifications" {
  cluster_name = "my-kafka-cluster"
  subject      = "notifications-value"
  schema_type  = "JSON"
  schema       = jsonencode({
    "$schema"    = "http://json-schema.org/draft-07/schema#"
    type         = "object"
    title        = "Notification"
    required     = ["id", "type", "message", "timestamp"]
    properties   = {
      id = {
        type = "string"
        format = "uuid"
      }
      type = {
        type = "string"
        enum = ["email", "sms", "push"]
      }
      recipient = {
        type = "string"
      }
      message = {
        type = "string"
        maxLength = 1000
      }
      timestamp = {
        type = "integer"
      }
      metadata = {
        type = "object"
        additionalProperties = true
      }
    }
  })
}

# Protobuf schema example
resource "axonops_schema" "sensor_data" {
  cluster_name = "my-kafka-cluster"
  subject      = "sensor-data-value"
  schema_type  = "PROTOBUF"
  schema       = <<-EOT
    syntax = "proto3";
    package com.example.sensors;

    message SensorReading {
      string sensor_id = 1;
      double value = 2;
      string unit = 3;
      int64 timestamp = 4;
      map<string, string> tags = 5;
    }
  EOT
}

# Key schema example
resource "axonops_schema" "user_events_key" {
  cluster_name = "my-kafka-cluster"
  subject      = "user-events-key"
  schema_type  = "AVRO"
  schema       = jsonencode({
    type      = "record"
    name      = "UserEventKey"
    namespace = "com.example.events"
    fields    = [
      {
        name = "user_id"
        type = "string"
      }
    ]
  })
}
