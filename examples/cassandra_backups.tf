# Cassandra Backup Examples

# Basic daily backup with local retention
resource "axonops_cassandra_backup" "daily" {
  cluster_name   = "my-cassandra-cluster"
  tag            = "daily-backup"
  datacenters    = ["dc1"]
  schedule       = true
  schedule_expr  = "0 1 * * *"  # Daily at 1 AM
  local_retention = "10d"
}

# Multi-datacenter backup with custom schedule
resource "axonops_cassandra_backup" "multi_dc" {
  cluster_name   = "my-cassandra-cluster"
  tag            = "multi-dc-backup"
  datacenters    = ["dc1", "dc2"]
  schedule       = true
  schedule_expr  = "0 3 * * 0"  # Weekly on Sunday at 3 AM
  local_retention = "30d"
  timeout        = "24h"
}

# Backup specific keyspaces and tables
resource "axonops_cassandra_backup" "selective" {
  cluster_name   = "my-cassandra-cluster"
  tag            = "selective-backup"
  datacenters    = ["dc1"]
  schedule       = true
  schedule_expr  = "0 2 * * *"  # Daily at 2 AM
  local_retention = "7d"

  keyspaces = ["my_keyspace", "analytics_keyspace"]
  tables    = ["my_keyspace.users", "my_keyspace.orders"]
}

# Backup with remote S3 storage
resource "axonops_cassandra_backup" "remote_s3" {
  cluster_name     = "my-cassandra-cluster"
  tag              = "s3-backup"
  datacenters      = ["dc1"]
  schedule         = true
  schedule_expr    = "0 0 * * *"  # Daily at midnight
  local_retention  = "3d"
  remote           = true
  remote_type      = "s3"
  remote_path      = "my-bucket/cassandra-backups"
  remote_retention = "90d"
  remote_config    = "access_key_id=AKIAIOSFODNN7EXAMPLE\nsecret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\nregion=us-east-1"
  transfers        = 4
  tps_limit        = 100
}

# Backup with remote SFTP storage
resource "axonops_cassandra_backup" "remote_sftp" {
  cluster_name     = "my-cassandra-cluster"
  tag              = "sftp-backup"
  datacenters      = ["dc1"]
  schedule         = true
  schedule_expr    = "0 4 * * *"  # Daily at 4 AM
  local_retention  = "5d"
  remote           = true
  remote_type      = "sftp"
  remote_path      = "/backups/cassandra"
  remote_retention = "60d"
  remote_config    = "host=backup-server.internal\nuser=backup\nkey_file=/etc/axonops/sftp_key"
}

# DSE cluster backup
resource "axonops_cassandra_backup" "dse" {
  cluster_name   = "my-dse-cluster"
  cluster_type   = "dse"
  tag            = "dse-daily-backup"
  datacenters    = ["dc1"]
  schedule       = true
  schedule_expr  = "0 1 * * *"
  local_retention = "14d"
  timeout        = "12h"
}

# Backup specific nodes only
resource "axonops_cassandra_backup" "node_specific" {
  cluster_name   = "my-cassandra-cluster"
  tag            = "node-backup"
  datacenters    = ["dc1"]
  schedule       = true
  schedule_expr  = "0 5 * * 6"  # Weekly on Saturday at 5 AM
  local_retention = "7d"
  nodes          = ["node-1-uuid", "node-2-uuid"]
}

# Read existing backup configuration
data "axonops_cassandra_backup" "existing" {
  cluster_name = "my-cassandra-cluster"
  tag          = "daily-backup"
}
