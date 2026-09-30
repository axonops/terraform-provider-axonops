# Basic topic with minimal configuration
resource "axonops_kafka_topic" "basic" {
  name               = "my-basic-topic"
  partitions         = 3
  replication_factor = 2
  cluster_name       = "my-kafka-cluster"
}

# Topic with custom configuration
resource "axonops_kafka_topic" "configured" {
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
