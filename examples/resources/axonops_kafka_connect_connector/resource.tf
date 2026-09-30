# File Source Connector
resource "axonops_kafka_connect_connector" "file_source" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "file-source-connector"

  config = {
    "connector.class" = "org.apache.kafka.connect.file.FileStreamSourceConnector"
    "tasks.max"       = "1"
    "file"            = "/var/log/application.log"
    "topic"           = "application-logs"
  }
}

# JDBC Source Connector
resource "axonops_kafka_connect_connector" "jdbc_source" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "postgres-source"

  config = {
    "connector.class"          = "io.confluent.connect.jdbc.JdbcSourceConnector"
    "tasks.max"                = "1"
    "connection.url"           = "jdbc:postgresql://localhost:5432/mydb"
    "connection.user"          = "dbuser"
    "connection.password"      = "dbpassword"
    "table.whitelist"          = "users,orders"
    "mode"                     = "incrementing"
    "incrementing.column.name" = "id"
    "topic.prefix"             = "postgres-"
  }
}
