# Kafka Metric Alert Rules
# Comprehensive alert rule set for Apache Kafka clusters monitored by AxonOps.
# All dashboard and chart names match AxonOps Kafka dashboard panels.
# Ported from the axonops-ansible-collection Kafka alert pack.


# ── Kafka Cluster Health ──────────────────────────────────────────────
# These three are the minimum every Kafka cluster must have.

resource "axonops_metric_alert_rule" "kafka_multiple_active_controllers" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Multiple Active Controllers"
  dashboard      = "Kafka Overview"
  chart          = "Active Controller"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "1m"

  annotations = {
    description = "More than one active controller — split-brain condition. Page immediately."
  }
}

resource "axonops_metric_alert_rule" "kafka_offline_partitions" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Offline Partitions"
  dashboard      = "Kafka Replication"
  chart          = "Offline Partitions"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "5m"

  annotations = {
    description = "Partitions with no active leader are unreadable and unwritable. Restore the failed broker."
  }
}

resource "axonops_metric_alert_rule" "kafka_unclean_leader_elections" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Unclean Leader Elections"
  dashboard      = "Kafka Overview"
  chart          = "Unclean Leader Elections Per Sec"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "1m"

  annotations = {
    description = "A partition leader was elected from a replica outside the ISR — potential data loss. Expected value is always 0."
  }
}


# ── Kafka Replication ─────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "kafka_under_replicated_partitions" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Under Replicated Partitions"
  dashboard      = "Kafka Replication"
  chart          = "Under Replicated Partitions"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "5m"

  annotations = {
    description = "Partitions have fewer ISR members than the replication factor. Data durability at risk. Investigate slow followers and network I/O."
  }
}

resource "axonops_metric_alert_rule" "kafka_under_min_isr_partitions" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Under Min ISR Partitions"
  dashboard      = "Kafka Replication"
  chart          = "Under Min ISR Partitions"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "1m"

  annotations = {
    description = "Partitions below min.insync.replicas. Producers using acks=all will receive NotEnoughReplicas and writes will fail."
  }
}

resource "axonops_metric_alert_rule" "kafka_isr_shrinks_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "ISR Shrinks High"
  dashboard      = "Kafka Replication"
  chart          = "IsrShrinks per Sec by Host"
  operator       = ">="
  warning_value  = 1
  critical_value = 5
  duration       = "5m"
  group_by       = ["host_id"]

  annotations = {
    description = "Replicas are falling out of sync. Expected value is 0 in steady state. Sustained shrinks indicate disk I/O or network degradation."
  }
}


# ── KRaft Controller ──────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "kafka_controller_metadata_error_rate" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Controller Metadata Error Rate"
  dashboard      = "Kafka Controller"
  chart          = "Metadata Error Rate"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "1m"

  annotations = {
    description = "Errors during metadata log processing on the controller. Investigate immediately."
  }
}

resource "axonops_metric_alert_rule" "kafka_raft_commit_latency_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Raft Commit Latency High"
  dashboard      = "Kafka Controller"
  chart          = "Commit Latency Avg"
  operator       = ">="
  warning_value  = 500
  critical_value = 2000
  duration       = "5m"

  annotations = {
    description = "Average commit latency for the raft log is high, indicating quorum throughput degradation."
  }
}

resource "axonops_metric_alert_rule" "kafka_preferred_replica_imbalance" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Preferred Replica Imbalance"
  dashboard      = "Kafka Overview"
  chart          = "Preferred Replica Imbalance"
  operator       = ">="
  warning_value  = 1
  critical_value = 10
  duration       = "15m"

  annotations = {
    description = "Partition leaders are not distributed optimally across brokers. Run preferred replica election or enable auto.leader.rebalance.enable."
  }
}


# ── Network & Request Processing ──────────────────────────────────────

resource "axonops_metric_alert_rule" "kafka_network_processor_idle_low" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Network Processor Avg Idle Percent Low"
  dashboard      = "Kafka Performance"
  chart          = "Network Processor Avg Idle Percent"
  operator       = "<"
  warning_value  = 30
  critical_value = 10
  duration       = "5m"

  annotations = {
    description = "Network I/O threads are saturated. Normal range is above 30%. Increase num.network.threads or scale the broker."
  }
}

resource "axonops_metric_alert_rule" "kafka_request_handler_idle_low" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Request Handler Avg Idle Percent Low"
  dashboard      = "Kafka Performance"
  chart          = "Request Handler Avg Idle Percent"
  operator       = "<"
  warning_value  = 30
  critical_value = 10
  duration       = "5m"

  annotations = {
    description = "Request handler threads are saturated. Normal range is above 30%. Increase num.io.threads or reduce per-broker partition count."
  }
}

resource "axonops_metric_alert_rule" "kafka_request_queue_size_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Request Queue Size High"
  dashboard      = "Kafka Performance"
  chart          = "Request Queue Size"
  operator       = ">="
  warning_value  = 200
  critical_value = 500
  duration       = "5m"

  annotations = {
    description = "Requests are backing up in the network queue, indicating handler saturation. Correlate with request handler idle percent."
  }
}


# ── Kafka Connect ─────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "kafka_connect_connector_failed" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Connector Failed"
  dashboard      = "Connect Workers"
  chart          = "Connector Failed"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "5m"

  annotations = {
    description = "One or more connectors are in FAILED state. Connect does NOT automatically restart failed connectors. Restart via REST API POST /connectors/{name}/restart."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_tasks_failed" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Tasks Failed"
  dashboard      = "Connect Workers"
  chart          = "Connector Failed Tasks"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "5m"

  annotations = {
    description = "One or more connector tasks have failed. Connect does NOT automatically restart failed tasks. Restart via REST API POST /connectors/{name}/tasks/{id}/restart."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_task_running_ratio_low" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Task Running Ratio Low"
  dashboard      = "Connect Tasks"
  chart          = "Connector Task Running Ratio"
  operator       = "<"
  warning_value  = 75
  critical_value = 50
  duration       = "5m"

  annotations = {
    description = "A task is spending more than half its time not running (paused or backpressured). Investigate downstream system performance."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_task_commit_success_rate_low" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Task Commit Success Rate Low"
  dashboard      = "Connect Tasks"
  chart          = "Connector Task Commit Success %"
  operator       = "<"
  warning_value  = 95
  critical_value = 80
  duration       = "5m"

  annotations = {
    description = "Offset commit success rate is low. Indicates consumer group instability or timeout issues. Increase session.timeout.ms or reduce max.poll.records."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_dlq_produce_failures" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect DLQ Produce Failures"
  dashboard      = "Connect Tasks"
  chart          = "Deadletter Produce Failures"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "5m"

  annotations = {
    description = "Failed writes to the dead letter queue mean error records are being silently lost. Check DLQ topic configuration and broker connectivity."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_record_errors_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Record Errors High"
  dashboard      = "Connect Tasks"
  chart          = "Record Errors"
  operator       = ">="
  warning_value  = 1
  critical_value = 100
  duration       = "5m"

  annotations = {
    description = "Record processing errors are accumulating. Investigate upstream data quality or schema compatibility."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_failed_authentication" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Failed Authentication"
  dashboard      = "Connect Overview"
  chart          = "Failed Authentication"
  operator       = ">="
  warning_value  = 1
  critical_value = 10
  duration       = "5m"

  annotations = {
    description = "Connect worker cannot authenticate to Kafka. Check SASL/SSL configuration and credentials."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_workers_rebalancing" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Workers Rebalancing"
  dashboard      = "Connect Overview"
  chart          = "Rebalances"
  operator       = ">="
  warning_value  = 1
  critical_value = 5
  duration       = "15m"

  annotations = {
    description = "Connect worker group is rebalancing repeatedly. A single rebalance is normal on worker join/leave; sustained rebalances indicate a failed worker or incompatible connector config."
  }
}

resource "axonops_metric_alert_rule" "kafka_connect_record_skipped_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Record Skipped High"
  dashboard      = "Connect Tasks"
  chart          = "Record Skipped"
  operator       = ">="
  warning_value  = 1
  critical_value = 100
  duration       = "5m"

  annotations = {
    description = "Records are being silently skipped due to errors with errors.tolerance=all. Data loss in progress. Review upstream data quality and connector error handling configuration."
  }
}


# ── Consumer Groups ───────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "kafka_consumer_group_lag_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Consumer Group Lag High"
  dashboard      = "Consumer Groups"
  chart          = "Consumer Group Lag"
  operator       = ">="
  warning_value  = 10000
  critical_value = 100000
  duration       = "5m"

  annotations = {
    description = "Consumer group is significantly behind the latest offset. Investigate consumer throughput, partition distribution, or broker pressure."
  }
}


# ── System ────────────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "kafka_cpu_usage_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "CPU usage per host"
  dashboard      = "System"
  chart          = "CPU usage per host"
  operator       = ">="
  warning_value  = 90
  critical_value = 99
  duration       = "1h"
  group_by       = ["host_id"]

  annotations = {
    description = "Detected High CPU usage"
  }
}

resource "axonops_metric_alert_rule" "kafka_cpu_underutilized" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "CPU is Underutilized"
  dashboard      = "System"
  chart          = "CPU usage per host"
  operator       = "<="
  warning_value  = 5
  critical_value = 1
  duration       = "1w"

  annotations = {
    description = "CPU load has been very low for 1 week"
  }
}

resource "axonops_metric_alert_rule" "kafka_disk_usage_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Disk % Usage $mountpoint"
  dashboard      = "System"
  chart          = "Disk % Usage $mountpoint"
  operator       = ">="
  warning_value  = 75
  critical_value = 90
  duration       = "12h"
  group_by       = ["host_id"]

  annotations = {
    description = "Detected High disk utilization"
  }
}

resource "axonops_metric_alert_rule" "kafka_io_wait_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Avg IO wait CPU per Host"
  dashboard      = "System"
  chart          = "Avg IO wait CPU per Host"
  operator       = ">="
  warning_value  = 20
  critical_value = 50
  duration       = "2h"
  group_by       = ["host_id"]

  annotations = {
    description = "Detected high Average IOWait"
  }
}

resource "axonops_metric_alert_rule" "kafka_memory_usage_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Used Memory Percentage"
  dashboard      = "System"
  chart          = "Used Memory Percentage"
  operator       = ">="
  warning_value  = 80
  critical_value = 90
  duration       = "1h"
  group_by       = ["host_id"]

  annotations = {
    description = "High memory utilization detected"
  }
}

resource "axonops_metric_alert_rule" "kafka_failed_auth_rate" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Failed Auth Rate"
  dashboard      = "Kafka Controller"
  chart          = "Failed Auth Rate"
  operator       = ">="
  warning_value  = 10
  critical_value = 50
  duration       = "15m"

  annotations = {
    description = "Failed authentication requests against Kafka. Check SASL/SSL configuration and client credentials."
  }
}
