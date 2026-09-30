# Check Kafka broker port
resource "axonops_healthcheck_tcp" "kafka_broker" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka Broker Port"
  tcp                   = "0.0.0.0:9092"
  interval              = "30s"
  timeout               = "10s"
  supported_agent_types = ["broker", "kraft-broker"]
}

# Check Kafka controller port
resource "axonops_healthcheck_tcp" "kafka_controller" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka Controller Port"
  tcp                   = "0.0.0.0:9093"
  interval              = "30s"
  timeout               = "10s"
  supported_agent_types = ["kraft-controller"]
}

# Check Schema Registry port
resource "axonops_healthcheck_tcp" "schema_registry" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Schema Registry Port"
  tcp                   = "0.0.0.0:8081"
  interval              = "1m"
  timeout               = "15s"
  supported_agent_types = ["schema-registry"]
}

# Check Kafka Connect port
resource "axonops_healthcheck_tcp" "kafka_connect" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka Connect Port"
  tcp                   = "0.0.0.0:8083"
  interval              = "1m"
  timeout               = "15s"
  supported_agent_types = ["all"]
}

# Check ZooKeeper client port
resource "axonops_healthcheck_tcp" "zookeeper" {
  cluster_name          = "my-kafka-cluster"
  name                  = "ZooKeeper Client Port"
  tcp                   = "0.0.0.0:2181"
  interval              = "30s"
  timeout               = "10s"
  supported_agent_types = ["zookeeper"]
}
