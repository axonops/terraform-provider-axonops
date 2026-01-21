# Healthcheck Examples

# TCP Healthchecks

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

# HTTP Healthchecks

# Check Schema Registry health endpoint
resource "axonops_healthcheck_http" "schema_registry_health" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Schema Registry Health"
  url                   = "http://localhost:8081/subjects"
  method                = "GET"
  expected_status       = 200
  interval              = "1m"
  timeout               = "30s"
  supported_agent_types = ["schema-registry"]
}

# Check Kafka Connect health endpoint
resource "axonops_healthcheck_http" "connect_health" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Kafka Connect Health"
  url                   = "http://localhost:8083/connectors"
  method                = "GET"
  expected_status       = 200
  interval              = "1m"
  timeout               = "30s"
  supported_agent_types = ["all"]
}

# Check custom application health with headers
resource "axonops_healthcheck_http" "app_health" {
  cluster_name          = "my-kafka-cluster"
  name                  = "Application Health"
  url                   = "http://localhost:8080/health"
  method                = "GET"
  expected_status       = 200
  interval              = "30s"
  timeout               = "10s"
  headers = {
    "Accept"        = "application/json"
    "Authorization" = "Bearer token123"
  }
  supported_agent_types = ["all"]
}

# POST request healthcheck
resource "axonops_healthcheck_http" "api_check" {
  cluster_name          = "my-kafka-cluster"
  name                  = "API Health Check"
  url                   = "http://localhost:8080/api/v1/healthcheck"
  method                = "POST"
  body                  = "{\"check\": \"deep\"}"
  expected_status       = 200
  interval              = "2m"
  timeout               = "30s"
  headers = {
    "Content-Type" = "application/json"
  }
  supported_agent_types = ["all"]
}

# Shell Healthchecks

# Check disk space
resource "axonops_healthcheck_shell" "disk_space" {
  cluster_name = "my-kafka-cluster"
  name         = "Disk Space Check"
  script       = "/usr/local/bin/check_disk_space.sh"
  shell        = "/bin/bash"
  interval     = "5m"
  timeout      = "30s"
}

# Check Kafka process
resource "axonops_healthcheck_shell" "kafka_process" {
  cluster_name = "my-kafka-cluster"
  name         = "Kafka Process Check"
  script       = "pgrep -f kafka.Kafka"
  interval     = "1m"
  timeout      = "10s"
}

# Check JVM heap usage
resource "axonops_healthcheck_shell" "jvm_heap" {
  cluster_name = "my-kafka-cluster"
  name         = "JVM Heap Check"
  script       = "/usr/local/bin/check_jvm_heap.sh"
  shell        = "/bin/bash"
  interval     = "2m"
  timeout      = "30s"
}

# Simple file existence check
resource "axonops_healthcheck_shell" "config_exists" {
  cluster_name = "my-kafka-cluster"
  name         = "Config File Check"
  script       = "test -f /etc/kafka/server.properties"
  interval     = "10m"
  timeout      = "5s"
}

# Check network connectivity
resource "axonops_healthcheck_shell" "network_check" {
  cluster_name = "my-kafka-cluster"
  name         = "Network Connectivity"
  script       = "ping -c 1 -W 2 kafka-broker-1.internal"
  interval     = "1m"
  timeout      = "10s"
}

# Custom monitoring script
resource "axonops_healthcheck_shell" "custom_monitor" {
  cluster_name = "my-kafka-cluster"
  name         = "Custom Monitoring Script"
  script       = "/opt/scripts/kafka-health-check.sh"
  shell        = "/bin/bash"
  interval     = "3m"
  timeout      = "1m"
}
