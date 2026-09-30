# Silence all alerts on a cluster for a fixed maintenance window
resource "axonops_silence" "maintenance_window" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  duration     = "2h"
  note         = "Scheduled maintenance: rolling AMI upgrade"
}

# Recurring silence for a specific datacenter, active every Sunday at 02:00
resource "axonops_silence" "weekly_backup_window" {
  cluster_name = "my-cassandra-cluster"
  cluster_type = "cassandra"
  duration     = "1h"
  is_recurring = true
  cron_expr    = "0 2 * * 0"
  datacenters  = ["dc1"]
  note         = "Weekly backup window - suppress disk I/O alerts"
}
