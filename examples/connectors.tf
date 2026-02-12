# Kafka Connect Connector Examples

# File Source Connector - reads from a file and produces to a topic
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

# File Sink Connector - consumes from a topic and writes to a file
resource "axonops_kafka_connect_connector" "file_sink" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "file-sink-connector"

  config = {
    "connector.class" = "org.apache.kafka.connect.file.FileStreamSinkConnector"
    "tasks.max"       = "1"
    "file"            = "/tmp/output.txt"
    "topics"          = "output-topic"
  }
}

# JDBC Source Connector - reads from a database
resource "axonops_kafka_connect_connector" "jdbc_source" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "postgres-source"

  config = {
    "connector.class"                    = "io.confluent.connect.jdbc.JdbcSourceConnector"
    "tasks.max"                          = "1"
    "connection.url"                     = "jdbc:postgresql://localhost:5432/mydb"
    "connection.user"                    = "dbuser"
    "connection.password"                = "dbpassword"
    "table.whitelist"                    = "users,orders"
    "mode"                               = "incrementing"
    "incrementing.column.name"           = "id"
    "topic.prefix"                       = "postgres-"
    "poll.interval.ms"                   = "5000"
  }
}

# Elasticsearch Sink Connector
resource "axonops_kafka_connect_connector" "elasticsearch_sink" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "elasticsearch-sink"

  config = {
    "connector.class"     = "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector"
    "tasks.max"           = "2"
    "topics"              = "user-events"
    "connection.url"      = "http://elasticsearch:9200"
    "type.name"           = "_doc"
    "key.ignore"          = "true"
    "schema.ignore"       = "true"
  }
}

# S3 Sink Connector
resource "axonops_kafka_connect_connector" "s3_sink" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "s3-sink"

  config = {
    "connector.class"                    = "io.confluent.connect.s3.S3SinkConnector"
    "tasks.max"                          = "4"
    "topics"                             = "events"
    "s3.bucket.name"                     = "my-kafka-backup"
    "s3.region"                          = "us-east-1"
    "flush.size"                         = "1000"
    "rotate.interval.ms"                 = "60000"
    "storage.class"                      = "io.confluent.connect.s3.storage.S3Storage"
    "format.class"                       = "io.confluent.connect.s3.format.json.JsonFormat"
    "partitioner.class"                  = "io.confluent.connect.storage.partitioner.TimeBasedPartitioner"
    "path.format"                        = "'year'=YYYY/'month'=MM/'day'=dd/'hour'=HH"
    "locale"                             = "en-US"
    "timezone"                           = "UTC"
  }
}

# Debezium MySQL CDC Connector
resource "axonops_kafka_connect_connector" "debezium_mysql" {
  cluster_name         = "my-kafka-cluster"
  connect_cluster_name = "my-connect-cluster"
  name                 = "mysql-cdc"

  config = {
    "connector.class"                    = "io.debezium.connector.mysql.MySqlConnector"
    "tasks.max"                          = "1"
    "database.hostname"                  = "mysql-server"
    "database.port"                      = "3306"
    "database.user"                      = "debezium"
    "database.password"                  = "dbz"
    "database.server.id"                 = "184054"
    "topic.prefix"                       = "mysql"
    "database.include.list"              = "inventory"
    "schema.history.internal.kafka.bootstrap.servers" = "kafka:9092"
    "schema.history.internal.kafka.topic"             = "schema-changes.inventory"
  }
}
