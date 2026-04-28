# Cassandra Backup Schedules
# Ported from the axonops-ansible-collection Cassandra backup examples.

# Daily backup to S3
resource "axonops_cassandra_backup" "s3_daily" {
  cluster_name     = var.cluster_name
  tag              = "scheduled backup"
  datacenters      = ["dc1"]
  remote_type      = "s3"
  remote_path      = "bucketname/path"
  local_retention  = "10d"
  remote_retention = "60d"
  timeout          = "10h"
  remote           = true
  schedule         = true
  schedule_expr    = "0 1 * * *"
  remote_config    = "region=eu-west-2\nacl=private"
}

# Snapshot a specific table to Azure Blob Storage
resource "axonops_cassandra_backup" "azure_table_snapshot" {
  cluster_name     = var.cluster_name
  tag              = "Snapshot appTable"
  datacenters      = ["dc1"]
  remote_type      = "azure"
  remote_path      = "foo"
  local_retention  = "10d"
  remote_retention = "30d"
  timeout          = "10h"
  remote           = true
  schedule         = false
  keyspaces        = ["appKeyspace"]
  tables           = ["appKeyspace.appTable"]
  remote_config    = "account=azure_storage_account_name\nuse_msi=true"
}
