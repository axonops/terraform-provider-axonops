variable "s3_remote_config" {
  description = "Remote storage configuration as key=value pairs separated by newlines"
  type        = string
  sensitive   = true
}

# Basic daily backup
resource "axonops_cassandra_backup" "daily" {
  cluster_name    = "my-cassandra-cluster"
  tag             = "daily-backup"
  datacenters     = ["dc1"]
  schedule        = true
  schedule_expr   = "0 1 * * *" # Daily at 1 AM
  local_retention = "10d"
}

# Backup with S3 remote storage
resource "axonops_cassandra_backup" "remote_s3" {
  cluster_name     = "my-cassandra-cluster"
  tag              = "s3-backup"
  datacenters      = ["dc1"]
  schedule         = true
  schedule_expr    = "0 0 * * *"
  local_retention  = "3d"
  remote           = true
  remote_type      = "s3"
  remote_path      = "my-bucket/cassandra-backups"
  remote_retention = "90d"
  remote_config    = var.s3_remote_config # e.g. "access_key_id=...\nsecret_access_key=...\nregion=us-east-1"
}

# Selective backup
resource "axonops_cassandra_backup" "selective" {
  cluster_name    = "my-cassandra-cluster"
  tag             = "selective-backup"
  datacenters     = ["dc1"]
  schedule        = true
  schedule_expr   = "0 2 * * *"
  local_retention = "7d"
  keyspaces       = ["my_keyspace"]
  tables          = ["my_keyspace.users", "my_keyspace.orders"]
}
