# Cassandra Metric Alert Rules
# Comprehensive alert rule set for Apache Cassandra clusters monitored by AxonOps.
# Ported from the axonops-ansible-collection Cassandra alert pack.
# Org-level rules (applied to all clusters) are followed by cluster-level table rules.


# ── Overview ──────────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "cassandra_down_count" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "DOWN count per node"
  dashboard      = "Overview"
  chart          = "UP vs Down endpoints"
  metric         = "max(cas_downendpoint_count) by (host_id)"
  operator       = ">="
  warning_value  = 1
  critical_value = 2
  duration       = "15m"

  annotations = {
    description = "Detected DOWN nodes"
  }
}


# ── System ────────────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "cassandra_cpu_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
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

resource "axonops_metric_alert_rule" "cassandra_cpu_underutilized" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
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

resource "axonops_metric_alert_rule" "cassandra_disk_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
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

resource "axonops_metric_alert_rule" "cassandra_io_wait_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
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

resource "axonops_metric_alert_rule" "cassandra_gc_duration_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "GC duration"
  dashboard      = "System"
  chart          = "GC duration"
  operator       = ">="
  warning_value  = 5000
  critical_value = 10000
  duration       = "2m"

  annotations = {
    description = "Detected high Garbage Collection cycle time — this is not necessarily the Stop-the-World pause time"
  }
}

resource "axonops_metric_alert_rule" "cassandra_memory_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Used Memory Percentage"
  dashboard      = "System"
  chart          = "Used Memory Percentage"
  operator       = ">="
  warning_value  = 95
  critical_value = 85
  duration       = "1h"
  group_by       = ["host_id"]

  annotations = {
    description = "High memory utilization detected"
  }
}

resource "axonops_metric_alert_rule" "cassandra_memory_underutilized" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Memory is Underutilized"
  dashboard      = "System"
  chart          = "Used Memory Percentage"
  operator       = "<="
  warning_value  = 20
  critical_value = 10
  duration       = "1w"

  annotations = {
    description = "Node memory has been very low for 1 week. Consider reducing memory space"
  }
}

resource "axonops_metric_alert_rule" "cassandra_ntp_offset_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "NTP offset (milliseconds)"
  dashboard      = "System"
  chart          = "NTP offset (milliseconds)"
  operator       = ">="
  warning_value  = 5
  critical_value = 10
  duration       = "15m"

  annotations = {
    description = "High NTP time offset detected"
  }
}


# ── Coordinator ───────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "cassandra_coord_read_latency_local_quorum" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Read Latency - LOCAL_QUORUM 99thPercentile"
  dashboard      = "Coordinator"
  chart          = "Coordinator Read $consistency Latency - $percentile"
  operator       = ">="
  warning_value  = 1000000
  critical_value = 2000000
  duration       = "15m"
  consistency    = ["LOCAL_QUORUM"]
  percentile     = ["99thPercentile"]

  annotations = {
    description = "Detected high LOCAL_QUORUM Coordinator Read 99thPercentile latency"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_read_latency_local_one" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Read Latency - LOCAL_ONE 99thPercentile"
  dashboard      = "Coordinator"
  chart          = "Coordinator Read $consistency Latency - $percentile"
  operator       = ">="
  warning_value  = 1000000
  critical_value = 2000000
  duration       = "15m"
  consistency    = ["LOCAL_ONE"]
  percentile     = ["99thPercentile"]

  annotations = {
    description = "Detected high LOCAL_ONE Coordinator Read 99thPercentile latency"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_range_read_latency" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Range Read Latency - 99thPercentile"
  dashboard      = "Coordinator"
  chart          = "Coordinator Range Read Request Latency - $percentile"
  operator       = ">="
  warning_value  = 1500000
  critical_value = 2500000
  duration       = "15m"
  percentile     = ["99thPercentile"]

  annotations = {
    description = "Detected high Coordinator Range Read 99thPercentile latency"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_write_latency_local_quorum" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Write Latency - LOCAL_QUORUM 99thPercentile"
  dashboard      = "Coordinator"
  chart          = "Coordinator Write $consistency Latency - $percentile"
  operator       = ">="
  warning_value  = 1000000
  critical_value = 1500000
  duration       = "15m"
  consistency    = ["LOCAL_QUORUM"]
  percentile     = ["99thPercentile"]

  annotations = {
    description = "Detected high LOCAL_QUORUM Coordinator Write 99thPercentile latency"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_write_latency_local_one" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Write Latency - LOCAL_ONE 99thPercentile"
  dashboard      = "Coordinator"
  chart          = "Coordinator Write $consistency Latency - $percentile"
  operator       = ">="
  warning_value  = 1000000
  critical_value = 1500000
  duration       = "15m"
  consistency    = ["LOCAL_ONE"]
  percentile     = ["99thPercentile"]

  annotations = {
    description = "Detected high LOCAL_ONE Coordinator Write 99thPercentile latency"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_read_timeouts" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Read Timeouts Per Second"
  dashboard      = "Coordinator"
  chart          = "Read Timeouts Per Second"
  operator       = ">="
  warning_value  = 2
  critical_value = 5
  duration       = "10m"

  annotations = {
    description = "Detected read Timeouts"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_write_timeouts" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Write Timeouts Per Second"
  dashboard      = "Coordinator"
  chart          = "Write Timeouts Per Second"
  operator       = ">="
  warning_value  = 2
  critical_value = 5
  duration       = "10m"

  annotations = {
    description = "Detected write timeouts"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_read_unavailables" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Read Unavailables Per Second"
  dashboard      = "Coordinator"
  chart          = "Read Unavailables Per Second"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "1h"
  group_by       = ["host_id"]

  annotations = {
    description = "Detected Read Unavailables"
  }
}

resource "axonops_metric_alert_rule" "cassandra_coord_write_unavailables" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Coordinator Write Unavailables Per Second"
  dashboard      = "Coordinator"
  chart          = "Write Unavailables Per Second"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "1h"
  group_by       = ["host_id"]

  annotations = {
    description = "Detected Write Unavailables"
  }
}

resource "axonops_metric_alert_rule" "cassandra_cas_read_failures" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "CAS Read Failures"
  dashboard      = "Coordinator"
  chart          = "CAS Read Failures"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "30m"

  annotations = {
    description = "Detected CAS Read Failures"
  }
}

resource "axonops_metric_alert_rule" "cassandra_cas_write_failures" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "CAS Write Failures"
  dashboard      = "Coordinator"
  chart          = "CAS Write Failures"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "30m"

  annotations = {
    description = "Detected CAS Write Failures"
  }
}

resource "axonops_metric_alert_rule" "cassandra_cas_read_unavailables" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "CAS Read Unavailables"
  dashboard      = "Coordinator"
  chart          = "CAS Read Unavailables"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "30m"

  annotations = {
    description = "Detected CAS Read Unavailables"
  }
}

resource "axonops_metric_alert_rule" "cassandra_cas_read_unfinished_commit" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "CAS Read Unfinished Commit"
  dashboard      = "Coordinator"
  chart          = "CAS Read Unfinished Commit"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "30m"

  annotations = {
    description = "Detected CAS Read Unfinished Commit"
  }
}

resource "axonops_metric_alert_rule" "cassandra_cas_write_unfinished_commit" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "CAS Write Unfinished Commit"
  dashboard      = "Coordinator"
  chart          = "CAS Write Unfinished Commit"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "30m"

  annotations = {
    description = "Detected CAS Write Unfinished Commit"
  }
}

resource "axonops_metric_alert_rule" "cassandra_cas_read_condition_not_met" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "CAS Read Condition Not Met"
  dashboard      = "Coordinator"
  chart          = "CAS Read Condition Not Met"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "30m"

  annotations = {
    description = "Detected CAS Read Condition Not Met"
  }
}

resource "axonops_metric_alert_rule" "cassandra_cas_write_condition_not_met" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "CAS Write Condition Not Met"
  dashboard      = "Coordinator"
  chart          = "CAS Write Condition Not Met"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "30m"

  annotations = {
    description = "Detected CAS Write Condition Not Met"
  }
}


# ── Dropped Messages ──────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "cassandra_dropped_mutations" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Dropped Mutations"
  dashboard      = "Dropped Messages"
  chart          = "Dropped Mutation per secs"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "30s"

  annotations = {
    description = "Detected dropped mutations"
  }
}

resource "axonops_metric_alert_rule" "cassandra_dropped_reads" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Dropped Reads"
  dashboard      = "Dropped Messages"
  chart          = "Dropped Read per secs"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "30s"

  annotations = {
    description = "Detected dropped read messages"
  }
}

resource "axonops_metric_alert_rule" "cassandra_dropped_hints" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Dropped Hints"
  dashboard      = "Dropped Messages"
  chart          = "Dropped Hints per secs"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"

  annotations = {
    description = "Detected dropped Hints"
  }
}


# ── Thread Pools ──────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "cassandra_blocked_tasks_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Total Blocked Tasks Rate"
  dashboard      = "Thread Pools"
  chart          = "Total Blocked Tasks Rate"
  operator       = ">="
  warning_value  = 64
  critical_value = 128
  duration       = "15m"

  annotations = {
    description = "Detected blocked threads"
  }
}

resource "axonops_metric_alert_rule" "cassandra_blocked_compaction_tasks" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Total Blocked Compaction Tasks Rate"
  dashboard      = "Thread Pools"
  chart          = "Total Blocked Tasks Rate"
  operator       = ">="
  warning_value  = 16
  critical_value = 32
  duration       = "15m"
  scope          = ["CompactionExecutor"]
  group_by       = ["scope"]

  annotations = {
    description = "Detected blocked compaction threads"
  }
}

resource "axonops_metric_alert_rule" "cassandra_pending_native_transport" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Pending Native_Transport_Requests"
  dashboard      = "Thread Pools"
  chart          = "Pending Tasks"
  operator       = ">="
  warning_value  = 500
  critical_value = 1000
  duration       = "15m"
  scope          = ["Native_Transport_Requests"]
  group_by       = ["scope"]

  annotations = {
    description = "Detected high Pending Native_Transport_Requests tasks"
  }
}

resource "axonops_metric_alert_rule" "cassandra_pending_read_stage" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Pending ReadStage"
  dashboard      = "Thread Pools"
  chart          = "Pending Tasks"
  operator       = ">="
  warning_value  = 500
  critical_value = 1000
  duration       = "15m"
  scope          = ["ReadStage"]
  group_by       = ["scope"]

  annotations = {
    description = "Detected high Pending ReadStage tasks"
  }
}

resource "axonops_metric_alert_rule" "cassandra_pending_mutation_stage" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Pending MutationStage"
  dashboard      = "Thread Pools"
  chart          = "Pending Tasks"
  operator       = ">="
  warning_value  = 500
  critical_value = 1000
  duration       = "15m"
  scope          = ["MutationStage"]
  group_by       = ["scope"]

  annotations = {
    description = "Detected high Pending MutationStage tasks"
  }
}

resource "axonops_metric_alert_rule" "cassandra_blocked_flush_writer" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Total Blocked Flush Writer Tasks Rate"
  dashboard      = "Thread Pools"
  chart          = "Total Blocked Tasks Rate"
  operator       = ">="
  warning_value  = 16
  critical_value = 32
  duration       = "10m"
  scope          = ["MemtableFlushWriter"]
  group_by       = ["scope"]

  annotations = {
    description = "Detected blocked MemtableFlushWriter threads"
  }
}

resource "axonops_metric_alert_rule" "cassandra_blocked_repair_task" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Total Blocked Repair_Task Tasks Rate"
  dashboard      = "Thread Pools"
  chart          = "Total Blocked Tasks Rate"
  operator       = ">="
  warning_value  = 64
  critical_value = 128
  duration       = "1h"
  scope          = ["Repair_Task"]
  group_by       = ["scope"]

  annotations = {
    description = "Detected blocked Repair_Task threads"
  }
}

resource "axonops_metric_alert_rule" "cassandra_pending_repair_task" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Total Pending Repair_Task Tasks Rate"
  dashboard      = "Thread Pools"
  chart          = "Pending Tasks"
  operator       = ">="
  warning_value  = 128
  critical_value = 256
  duration       = "3h"
  scope          = ["Repair_Task"]
  group_by       = ["scope"]

  annotations = {
    description = "Too many pending repair threads"
  }
}

resource "axonops_metric_alert_rule" "cassandra_pending_compaction_tasks" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Pending TP Compaction Tasks"
  dashboard      = "Compactions"
  chart          = "Pending TP Compaction Tasks"
  operator       = ">="
  warning_value  = 200
  critical_value = 500
  duration       = "4h"

  annotations = {
    description = "Detected high Compaction tasks in queue to be processed"
  }
}


# ── Entropy ───────────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "cassandra_hints_created_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Total Hints Created By Each Node"
  dashboard      = "Entropy"
  chart          = "Total Hints Created By Each Node"
  operator       = ">="
  warning_value  = 50
  critical_value = 100
  duration       = "2h"

  annotations = {
    description = "Hints have been created for over 2 hours."
  }
}


# ── Cache ─────────────────────────────────────────────────────────────

resource "axonops_metric_alert_rule" "cassandra_keycache_hitrate_low" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "KeyCache HitRate Per node"
  dashboard      = "Cache"
  chart          = "KeyCache HitRate Per Node"
  operator       = "<="
  warning_value  = 0.03
  critical_value = 0.01
  duration       = "20m"

  annotations = {
    description = "KeyCache HitRate too low"
  }
}


# ── Security ──────────────────────────────────────────────────────────

# Security panels in the Cassandra dashboard are events_timeline (audit log
# events), not metric panels — they cannot be used with axonops_metric_alert_rule.
# The provider has no metric query to extract from. Equivalent rules have been
# moved to log_alerts.tf as axonops_log_alert_rule resources targeting the
# Cassandra audit log.


# ── Table-Level Alerts (cluster-specific — update keyspace/scope for your tables) ──

resource "axonops_metric_alert_rule" "cassandra_tombstones_scanned_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Tombstones Scanned per Table - 99thPercentile"
  dashboard      = "Table"
  chart          = "Tombstones Scanned per Table - $percentile"
  operator       = ">="
  warning_value  = 500
  critical_value = 1000
  duration       = "3h"
  keyspace       = ["system"]
  scope          = ["peers"]
  percentile     = ["99thPercentile"]

  annotations = {
    description = "Detected a high number of tombstones scanned during read"
  }
}

resource "axonops_metric_alert_rule" "cassandra_max_partition_size_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Max Partition Size"
  dashboard      = "Table"
  chart          = "Max Table Partition Size per $groupBy"
  operator       = ">="
  warning_value  = 104857600
  critical_value = 209715200
  duration       = "6h"
  keyspace       = ["system"]
  scope          = ["peers"]
  group_by       = ["host_id"]

  annotations = {
    description = "Detected high partition size"
  }
}

resource "axonops_metric_alert_rule" "cassandra_bloom_filter_false_positive_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Bloom Filter False Positive Ratio"
  dashboard      = "Table"
  chart          = "Bloom Filter False Positive Ratio"
  operator       = ">="
  warning_value  = 0.03
  critical_value = 0.05
  duration       = "3h"
  keyspace       = ["system"]
  scope          = ["peers"]

  annotations = {
    description = "Detected high Bloom Filter False Positive Ratio"
  }
}

resource "axonops_metric_alert_rule" "cassandra_sstables_read_per_query_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "SSTables Read Per Query - 99thPercentile"
  dashboard      = "Table"
  chart          = "SSTables Per Read - $percentile"
  operator       = ">="
  warning_value  = 15
  critical_value = 50
  duration       = "2h"
  keyspace       = ["system"]
  scope          = ["peers"]
  percentile     = ["99thPercentile"]

  annotations = {
    description = "Detected high SSTables read per query"
  }
}

resource "axonops_metric_alert_rule" "cassandra_live_sstables_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Live SSTables"
  dashboard      = "Table"
  chart          = "Max Live SSTables per Table per $groupBy"
  operator       = ">="
  warning_value  = 100
  critical_value = 300
  duration       = "6h"
  keyspace       = ["system"]
  scope          = ["peers"]
  group_by       = ["host_id"]

  annotations = {
    description = "Detected high SSTables count"
  }
}

resource "axonops_metric_alert_rule" "cassandra_speculative_retries_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Speculative Retries"
  dashboard      = "Table"
  chart          = "SpeculativeRetries By Node For Table Reads Per Second"
  operator       = ">="
  warning_value  = 200
  critical_value = 500
  duration       = "2h"
  keyspace       = ["system"]
  scope          = ["peers"]
  group_by       = ["host_id"]

  annotations = {
    description = "Detected high Speculative Retries"
  }
}
