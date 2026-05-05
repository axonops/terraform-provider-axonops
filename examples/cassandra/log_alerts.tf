# Cassandra Log Alert Rules
# Comprehensive log-based alert rules for Apache Cassandra clusters.
# All rules target /var/log/cassandra/system.log unless noted otherwise.
# Ported from the axonops-ansible-collection Cassandra alert pack.

resource "axonops_log_alert_rule" "cassandra_node_down" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Node Down"
  content        = "is now DOWN"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 5
  duration       = "5m"
  present        = true
  description    = "Detected node down"
}

resource "axonops_log_alert_rule" "cassandra_unsupported_protocol" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Unsupported Protocol"
  content        = "Invalid or unsupported protocol version"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 30
  duration       = "5m"
  present        = true
  description    = "Detected clients connecting with invalid or unsupported protocol version"
}

resource "axonops_log_alert_rule" "cassandra_repair_not_in_progress" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Repair are not in progress"
  content        = "Repair-Task"
  source         = "/var/log/cassandra/system.log"
  operator       = "<"
  warning_value  = 1
  critical_value = 1
  duration       = "24h"
  present        = true
  description    = "Detected no repair has been seen in the last 24h"
}

resource "axonops_log_alert_rule" "cassandra_tls_handshake_failure" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "TLS failed to handshake with peer"
  content        = "Failed to handshake with peer"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 50
  critical_value = 100
  duration       = "5m"
  present        = true
  description    = "Detected TLS handshake error with peer"
}

resource "axonops_log_alert_rule" "cassandra_dropping_gossip_message" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Dropping gossip message"
  content        = "dropping message of type GOSSIP"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "1m"
  present        = true
  description    = "Detected gossip message drops"
}

resource "axonops_log_alert_rule" "cassandra_failed_stream_session" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Failed stream session"
  content        = "failed stream session"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "5m"
  present        = true
  description    = "Detected stream session failure"
}

resource "axonops_log_alert_rule" "cassandra_corrupt_sstable" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Corrupt SSTable"
  content        = "Corrupt sstable"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10s"
  present        = true
  description    = "Detected SSTable file corruption"
}

resource "axonops_log_alert_rule" "cassandra_anticompaction" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Anticompaction"
  content        = "Starting anticompaction"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1000
  duration       = "5m"
  present        = true
  description    = "Detected anticompaction — possibly triggered by an incremental repair"
}

resource "axonops_log_alert_rule" "cassandra_jna_not_found" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "JNA Check"
  content        = "JNA not found"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "30s"
  present        = true
  description    = "Missing JNA"
}

resource "axonops_log_alert_rule" "cassandra_no_space_for_compaction" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Unable to compact due to disk space"
  content        = "Not enough space for compaction"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "15m"
  present        = true
  description    = "Unable to compact due to disk space"
}

resource "axonops_log_alert_rule" "cassandra_networking_buffer_pool_full" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Maximum memory usage reached for networking buffer pool"
  content        = "+Messaging-EventLoop +NoSpamLogger +cannot +allocate +chunk"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 3
  critical_value = 20
  duration       = "60m"
  present        = true
  description    = "Maximum memory usage reached; networking_cache_size needs to be increased"
}

resource "axonops_log_alert_rule" "cassandra_jvm_memory_lock_enomem" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Unable to lock JVM memory (ENOMEM)"
  content        = "Unable to lock JVM memory (ENOMEM)"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "15m"
  present        = true
  description    = "Unable to lock JVM memory (ENOMEM); increase RLIMIT_MEMLOCK"
}

resource "axonops_log_alert_rule" "cassandra_unknown_mlockall_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Unknown mlockall error"
  content        = "Unknown mlockall error"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "15m"
  present        = true
  description    = "Unknown mlockall error"
}

resource "axonops_log_alert_rule" "cassandra_unsupported_os" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "the current operating system is unsupported"
  content        = "the current operating system"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "15m"
  present        = true
  description    = "The current operating system is unsupported by Cassandra"
}

resource "axonops_log_alert_rule" "cassandra_obsolete_jna" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Obsolete version of JNA present"
  content        = "Obsolete version of JNA present"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "15m"
  present        = true
  description    = "Obsolete version of JNA present; unable to read errno. Upgrade to JNA 3.2.7 or later"
}

resource "axonops_log_alert_rule" "cassandra_writing_large_partition" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Writing large partition"
  content        = "Writing large partition"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 500
  duration       = "15m"
  present        = true
  description    = "Cassandra is writing a large partition on disk. This can create issues with reads and repairs. Review the schema."
}

resource "axonops_log_alert_rule" "cassandra_dropped_messages_overloaded" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Dropped messages due to overloaded node during repairs"
  content        = "LARGE_MESSAGES-[a-zA-Z0-9]+ overloaded; dropping"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 500
  duration       = "15m"
  present        = true
  description    = "Cassandra had to drop messages due to repairs overloading the node."
}

resource "axonops_log_alert_rule" "cassandra_jemalloc_not_preloaded" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Jemalloc shared library could not be preloaded"
  content        = "jemalloc shared library could not be preloaded to speed up memory allocations"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 500
  duration       = "15m"
  present        = true
  description    = "Jemalloc shared library could not be preloaded. This can affect performance."
}

resource "axonops_log_alert_rule" "cassandra_prepared_statement_cache_full" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Server Prepared Statement Cache Size"
  content        = "WARN .*prepared statements discarded"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 2
  critical_value = 100
  duration       = "10m"
  present        = true
  description    = "Prepared statements discarded because cache limit reached."
}


# ── Security (audit log) ──────────────────────────────────────────────
# Cassandra audit log path assumes default axon-cassandra-agent config.
# Adjust `source` if your cluster uses a different audit log location.

resource "axonops_log_alert_rule" "cassandra_failed_authentications" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Failed Authentications"
  content        = "category:AUTH type:LOGIN_ERROR"
  source         = "/var/log/cassandra/audit/audit.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 20
  duration       = "3m"
  present        = true
  description    = "Failed authentication in Cassandra"
}

resource "axonops_log_alert_rule" "cassandra_jmx_connections_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "JMX connections"
  content        = "JMX connection"
  source         = "/var/log/cassandra/system.log"
  operator       = ">="
  warning_value  = 10
  critical_value = 100
  duration       = "3m"
  present        = true
  description    = "JMX connections in Cassandra"
}

resource "axonops_log_alert_rule" "cassandra_failed_authorizations" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "Failed Authorizations"
  content        = "category:AUTH type:UNAUTHORIZED_ATTEMPT"
  source         = "/var/log/cassandra/audit/audit.log"
  operator       = ">="
  warning_value  = 5
  critical_value = 20
  duration       = "3m"
  present        = true
  description    = "Failed authorizations in Cassandra"
}

resource "axonops_log_alert_rule" "cassandra_ddl_queries_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "DDL queries"
  content        = "category:DDL"
  source         = "/var/log/cassandra/audit/audit.log"
  operator       = ">="
  warning_value  = 5
  critical_value = 100
  duration       = "3m"
  present        = true
  description    = "DDL queries run in Cassandra"
}

resource "axonops_log_alert_rule" "cassandra_dcl_queries_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "DCL queries"
  content        = "category:DCL"
  source         = "/var/log/cassandra/audit/audit.log"
  operator       = ">="
  warning_value  = 5
  critical_value = 100
  duration       = "3m"
  present        = true
  description    = "DCL queries run in Cassandra"
}

resource "axonops_log_alert_rule" "cassandra_dml_queries_high" {
  cluster_name   = var.cluster_name
  cluster_type   = "cassandra"
  name           = "DML queries"
  content        = "category:DML"
  source         = "/var/log/cassandra/audit/audit.log"
  operator       = ">="
  warning_value  = 5
  critical_value = 100
  duration       = "3m"
  present        = true
  description    = "DML queries in Cassandra"
}
