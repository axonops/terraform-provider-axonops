# All configs of broker 1
data "axonops_kafka_broker_config" "broker1" {
  cluster_name = "my-kafka-cluster"
  broker_id    = 1
}

output "broker1_retention_hours" {
  value = data.axonops_kafka_broker_config.broker1.values["log.retention.hours"]
}

# Only selected configs, with metadata
data "axonops_kafka_broker_config" "replication" {
  cluster_name = "my-kafka-cluster"
  broker_id    = 1
  config_names = ["default.replication.factor", "min.insync.replicas"]
}

output "replication_configs" {
  value = [
    for c in data.axonops_kafka_broker_config.replication.configs :
    "${c.name}=${c.value} (${c.source})"
  ]
}
