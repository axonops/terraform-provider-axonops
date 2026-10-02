variable "commitlog_s3_remote_config" {
  description = "rclone-style S3 remote configuration (key = value lines)"
  type        = string
  sensitive   = true
}

# Archive commitlogs to a directory on each node
resource "axonops_cassandra_commitlog_settings" "local" {
  cluster_name     = "my-cassandra-cluster"
  datacenters      = ["dc1"]
  remote_type      = "local"
  remote_path      = "/var/lib/cassandra/commitlog_archive"
  remote_retention = "7d"
}

# Archive commitlogs of two datacenters to S3 using instance credentials
resource "axonops_cassandra_commitlog_settings" "s3_instance_role" {
  cluster_name     = "my-cassandra-cluster"
  datacenters      = ["dc2", "dc3"]
  remote_type      = "s3"
  remote_path      = "my-bucket/commitlogs"
  remote_retention = "30d"
  remote_config    = "provider = AWS\nregion = eu-west-1\nenv_auth = true"
}

# Archive to S3 with explicit credentials and a bandwidth cap
resource "axonops_cassandra_commitlog_settings" "s3_keys" {
  cluster_name     = "my-cassandra-cluster"
  datacenters      = ["dc4"]
  remote_type      = "s3"
  remote_path      = "my-bucket/commitlogs"
  remote_retention = "90d"
  remote_config    = var.commitlog_s3_remote_config # e.g. "provider = AWS\nregion = us-east-1\nenv_auth = false\naccess_key_id = ...\nsecret_access_key = ..."
  timeout          = "2h"
  transfers        = 4
  bw_limit         = "20M"
}
