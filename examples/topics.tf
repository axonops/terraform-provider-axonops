# Kafka Topic Examples

# Basic topic with minimal configuration
resource "axonops_topic_resource" "basic" {
  name               = "my-basic-topic"
  partitions         = 3
  replication_factor = 2
  cluster_name       = "my-kafka-cluster"
}

# Topic with custom configuration
resource "axonops_topic_resource" "configured" {
  name               = "my-configured-topic"
  partitions         = 6
  replication_factor = 3
  cluster_name       = "my-kafka-cluster"

  # Topic configurations (use underscores instead of dots)
  config = {
    cleanup_policy      = "compact"
    retention_ms        = "604800000"  # 7 days
    segment_bytes       = "1073741824" # 1GB
    min_insync_replicas = "2"
  }
}

# Topic for event streaming with delete policy
resource "axonops_topic_resource" "events" {
  name               = "user-events"
  partitions         = 12
  replication_factor = 3
  cluster_name       = "my-kafka-cluster"

  config = {
    cleanup_policy       = "delete"
    retention_ms         = "259200000"    # 3 days
    delete_retention_ms  = "86400000"     # 1 day
    max_message_bytes    = "1048576"      # 1MB
  }
}

# Compacted topic for state storage
resource "axonops_topic_resource" "state_store" {
  name               = "user-profiles"
  partitions         = 6
  replication_factor = 3
  cluster_name       = "my-kafka-cluster"

  config = {
    cleanup_policy               = "compact"
    min_cleanable_dirty_ratio    = "0.1"
    segment_ms                   = "3600000"  # 1 hour
    delete_retention_ms          = "86400000" # 1 day
  }
}
