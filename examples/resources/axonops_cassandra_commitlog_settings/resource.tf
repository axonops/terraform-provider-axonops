variable "commitlog_s3_remote_config" {
  description = "Remote storage configuration as key=value pairs separated by newlines"
  type        = string
  sensitive   = true
}

# Archive commitlogs of dc1 to a directory on each node
resource "axonops_cassandra_commitlog_settings" "dc1" {
  cluster_name     = "my-cassandra-cluster"
  datacenter       = "dc1"
  remote_type      = "local"
  remote_path      = "/var/lib/cassandra/commitlog_archive"
  remote_retention = "7d"
}

# Archive commitlogs of dc2 to S3 using instance credentials
resource "axonops_cassandra_commitlog_settings" "dc2" {
  cluster_name     = "my-cassandra-cluster"
  datacenter       = "dc2"
  remote_type      = "s3"
  remote_path      = "my-bucket/commitlogs/dc2"
  remote_retention = "30d"
  remote_config    = "provider = AWS\nregion = eu-west-1\nenv_auth = true"
}

# Archive every datacenter to S3 with explicit credentials and a bandwidth cap
resource "axonops_cassandra_commitlog_settings" "all" {
  for_each = toset(["dc3", "dc4"])

  cluster_name     = "my-cassandra-cluster"
  datacenter       = each.key
  remote_type      = "s3"
  remote_path      = "my-bucket/commitlogs/${each.key}"
  remote_retention = "90d"
  remote_config    = var.commitlog_s3_remote_config # e.g. "provider = AWS\nregion = us-east-1\nenv_auth = false\naccess_key_id = ...\nsecret_access_key = ..."
  timeout          = "2h"
  transfers        = 4
  bw_limit         = "20M"
}
