# Kafka Log Alert Rules
# Comprehensive log-based alert rules for Apache Kafka clusters monitored by AxonOps.
# Covers: Kafka Broker, KRaft Controller, and Kafka Connect log streams.
# Ported from the axonops-ansible-collection Kafka alert pack.


# ── Kafka Broker — Startup & Availability ────────────────────────────

resource "axonops_log_alert_rule" "kafka_broker_startup_failure" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker Startup Failure"
  content        = "Fatal error during KafkaServer startup"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Broker failed to start. Check port conflicts, config errors, or permissions. Verify server.properties and listener ports."
}

resource "axonops_log_alert_rule" "kafka_broker_jvm_oom" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker JVM OOM"
  content        = "OutOfMemoryError"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "JVM heap exhausted. Increase -Xmx, review message sizes, check for memory leaks in custom interceptors or serialisers."
}

resource "axonops_log_alert_rule" "kafka_broker_fatal_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker FATAL Error"
  content        = "] FATAL "
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Fatal condition causing broker abort. Investigate immediately; check full stack trace."
}

resource "axonops_log_alert_rule" "kafka_storage_exception" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Kafka Storage Exception"
  content        = "KafkaStorageException"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Disk-level failure. Check disk health, SMART status, and filesystem mount state."
}

resource "axonops_log_alert_rule" "kafka_data_directory_failure" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Data Directory Failure"
  content        = "Failed to create or validate data directory"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Kafka cannot access a log directory. Verify directory exists, permissions are correct, and the filesystem is mounted and writable."
}

resource "axonops_log_alert_rule" "kafka_log_flush_io_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Log Flush IO Error"
  content        = "Error while flushing log"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Typically java.io.IOException — No space left on device. Expand storage, reduce retention, or delete unused topics."
}

resource "axonops_log_alert_rule" "kafka_disk_lock_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Disk Lock Error"
  content        = "Disk error while locking directory"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Another process is holding the data directory lock or the filesystem is read-only. Ensure no duplicate Kafka processes are running."
}


# ── Kafka Broker — Replication & Partition Health ─────────────────────

resource "axonops_log_alert_rule" "kafka_broker_under_replicated_partitions" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker Under Replicated Partitions"
  content        = "under replicated"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 5
  duration       = "5m"
  description    = "Partitions have fewer in-sync replicas than the replication factor. Risk of data loss if another broker fails."
}

resource "axonops_log_alert_rule" "kafka_isr_shrunk" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "ISR Shrunk"
  content        = "ISR shrunk"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 3
  critical_value = 20
  duration       = "5m"
  description    = "A replica fell behind and was removed from the ISR. Check the lagging broker's disk I/O, network latency, and resource utilisation."
}

resource "axonops_log_alert_rule" "kafka_isr_shrunk_state_change" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "ISR Shrunk (state-change log)"
  content        = "Shrinking ISR from"
  source         = "/var/log/kafka/state-change.log"
  operator       = ">="
  warning_value  = 3
  critical_value = 20
  duration       = "5m"
  description    = "ISR shrink logged in state-change log. Indicates replication performance issues."
}

resource "axonops_log_alert_rule" "kafka_offline_partition_log" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Offline Partition"
  content        = "OfflinePartition"
  source         = "/var/log/kafka/state-change.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "A partition has no active leader. Restore the failed broker or trigger partition reassignment."
}


# ── Kafka Broker — Security & Authentication ──────────────────────────

resource "axonops_log_alert_rule" "kafka_authorization_failure" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Authorization Failure"
  content        = "Authorization failed"
  source         = "/var/log/kafka/kafka-authorizer.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 10
  duration       = "5m"
  description    = "A client request was denied by the ACL authoriser. If unexpected, investigate potential misconfiguration or security breach."
}

resource "axonops_log_alert_rule" "kafka_authentication_failure" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Authentication Failure"
  content        = "Failed authentication"
  source         = "/var/log/kafka/server.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 20
  duration       = "5m"
  description    = "Client failed SASL/mTLS authentication. Check client credentials, SASL configuration, and SCRAM user store."
}


# ── Kafka Broker — JVM GC ─────────────────────────────────────────────

resource "axonops_log_alert_rule" "kafka_broker_full_gc" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker Full GC Event"
  content        = "Full GC"
  source         = "/var/log/kafka/kafkaServer-gc.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 3
  duration       = "15m"
  description    = "Stop-the-world full GC occurred. If frequent, increase heap size. Kafka recommends G1GC with -XX:MaxGCPauseMillis=20."
}

resource "axonops_log_alert_rule" "kafka_broker_long_gc_pause" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker Long GC Pause"
  content        = "GC.* pause .* [0-9]{4,}ms"
  source         = "/var/log/kafka/kafkaServer-gc.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 3
  duration       = "15m"
  description    = "GC pause exceeding 1s can cause consumer lag spikes and request timeouts. Tune JVM heap (4-8 GB recommended)."
}

resource "axonops_log_alert_rule" "kafka_broker_g1gc_mark_abort" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker G1GC Mark Abort"
  content        = "concurrent-mark-abort"
  source         = "/var/log/kafka/kafkaServer-gc.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "5m"
  description    = "G1GC marking cycle aborted due to heap pressure, often preceding an OOM. Review heap sizing immediately."
}

resource "axonops_log_alert_rule" "kafka_broker_gc_oom" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Broker GC OOM"
  content        = "OutOfMemoryError"
  source         = "/var/log/kafka/kafkaServer-gc.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "JVM heap exhausted as detected in GC log. Increase -Xmx immediately."
}


# ── KRaft Controller — Quorum & Leader Election ───────────────────────

resource "axonops_log_alert_rule" "kafka_controller_no_quorum_leader" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Controller No Quorum Leader"
  content        = "the leader is (none)"
  source         = "/var/log/kafka/controller.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "The KRaft quorum has no elected leader. The cluster cannot process metadata changes. Check connectivity between all controller nodes."
}

resource "axonops_log_alert_rule" "kafka_controller_fatal_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Controller FATAL Error"
  content        = "] FATAL "
  source         = "/var/log/kafka/controller.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Fatal condition in the controller process. Investigate the full stack trace immediately."
}

resource "axonops_log_alert_rule" "kafka_controller_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Controller ERROR"
  content        = "] ERROR "
  source         = "/var/log/kafka/controller.log"
  operator       = ">="
  warning_value  = 3
  critical_value = 20
  duration       = "5m"
  description    = "Error in the controller log. Any error indicates a significant operational event."
}

resource "axonops_log_alert_rule" "kafka_controller_oom" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Controller OOM"
  content        = "OutOfMemoryError"
  source         = "/var/log/kafka/controller.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Controller JVM heap exhausted. Increase controller heap — larger clusters require more memory for in-memory metadata."
}

resource "axonops_log_alert_rule" "kafka_controller_quorum_instability" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Controller Quorum Instability"
  content        = "Completed transition to Unattached"
  source         = "/var/log/kafka/controller.log"
  operator       = ">="
  warning_value  = 3
  critical_value = 20
  duration       = "15m"
  description    = "Controller is frequently losing its quorum attachment. May indicate network instability between controller nodes. A few occurrences per week can be normal but sustained frequency is not."
}

resource "axonops_log_alert_rule" "kafka_controller_metadata_load_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Controller Metadata Load Error"
  content        = "MetadataLoadError"
  source         = "/var/log/kafka/controller.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "The BrokerMetadataListener encountered errors loading the metadata log. Data integrity risk; investigate immediately."
}


# ── Kafka Connect — Worker Health ─────────────────────────────────────

resource "axonops_log_alert_rule" "kafka_connect_worker_fatal_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Worker FATAL Error"
  content        = "] FATAL "
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "The Connect worker process has encountered a fatal error. Restart the worker and investigate the root cause."
}

resource "axonops_log_alert_rule" "kafka_connect_worker_startup_failure" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Worker Startup Failure"
  content        = "Failed to start worker"
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "Worker failed to start, typically due to configuration errors. Validate connect-distributed.properties."
}


# ── Kafka Connect — Connector & Task State ────────────────────────────

resource "axonops_log_alert_rule" "kafka_connect_sink_task_failed" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Sink Task FAILED"
  content        = "WorkerSinkTask.*FAILED"
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "A sink connector task has transitioned to FAILED state. Connect does NOT automatically restart failed tasks. Restart via REST API POST /connectors/{name}/tasks/{id}/restart."
}

resource "axonops_log_alert_rule" "kafka_connect_source_task_failed" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Source Task FAILED"
  content        = "WorkerSourceTask.*FAILED"
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "A source connector task has transitioned to FAILED state. Connect does NOT automatically restart failed tasks. Restart via REST API POST /connectors/{name}/tasks/{id}/restart."
}

resource "axonops_log_alert_rule" "kafka_connect_offset_commit_failure" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Offset Commit Failure"
  content        = "Failed to commit offsets"
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 5
  duration       = "5m"
  description    = "Offset commit failed, typically due to consumer group rebalance timeout. Increase session.timeout.ms or reduce max.poll.records."
}

resource "axonops_log_alert_rule" "kafka_connect_deserialization_error" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Deserialization Error"
  content        = "DeserializationException"
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 10
  duration       = "5m"
  description    = "Schema or serialisation mismatch. Check schema registry configuration and producer schema compatibility settings."
}

resource "axonops_log_alert_rule" "kafka_connect_rest_api_unreachable" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Worker REST API Unreachable"
  content        = "Connection refused.*8083"
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 1
  duration       = "10m"
  description    = "The Connect worker REST API is unreachable — indicates a worker crash or network issue."
}

resource "axonops_log_alert_rule" "kafka_connect_broker_connection_loss" {
  cluster_name   = var.cluster_name
  cluster_type   = "kafka"
  name           = "Connect Broker Connection Loss"
  content        = "Disconnected from node"
  source         = "/var/log/kafka/connect.log"
  operator       = ">="
  warning_value  = 1
  critical_value = 5
  duration       = "5m"
  description    = "Connect worker lost connectivity to a Kafka broker. Check broker health and network path between Connect and Kafka."
}
