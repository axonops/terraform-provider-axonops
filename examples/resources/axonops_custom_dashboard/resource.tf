# Kafka throughput dashboard with a host filter
resource "axonops_custom_dashboard" "kafka_throughput" {
  cluster_type = "kafka"
  cluster_name = "my-kafka-cluster"
  name         = "Kafka Throughput"

  filters = [{
    name  = "host"
    label = "Broker"
    type  = "query"
    multi = true
    query = "host_id"
  }]

  panels = [
    {
      # A row groups the panels that follow it.
      title  = "Throughput"
      type   = "row"
      layout = { x = 0, y = 0, w = 18, h = 1 }
    },
    {
      title = "Messages in per second"
      type  = "line-chart"
      details = jsonencode({
        queries = [{
          query  = "sum by (host_id) (kaf_BrokerTopicMetrics_MessagesInPerSec{function='OneMinuteRate',host_id=~'$host'})"
          legend = "{{host_id}}"
        }]
      })
      layout = { x = 0, y = 1, w = 9, h = 6 }
    },
    {
      title = "Bytes in per second"
      type  = "line-chart"
      details = jsonencode({
        queries = [{
          query  = "sum by (host_id) (kaf_BrokerTopicMetrics_BytesInPerSec{function='OneMinuteRate',host_id=~'$host'})"
          legend = "{{host_id}}"
        }]
      })
      layout = { x = 9, y = 1, w = 9, h = 6 }
    },
  ]
}

# Cassandra dashboard with a fixed list of data centres
resource "axonops_custom_dashboard" "cassandra_latency" {
  cluster_type = "cassandra"
  cluster_name = "my-cassandra-cluster"
  name         = "Read Latency"

  filters = [{
    name   = "dc"
    label  = "Data centre"
    type   = "custom"
    values = "dc1,dc2"
  }]

  panels = [{
    title = "Coordinator read latency p99"
    type  = "line-chart"
    details = jsonencode({
      queries = [{
        query  = "cas_ClientRequest_Latency{scope='Read',function='99thPercentile',dc=~'$dc'}"
        legend = "{{host_id}}"
      }]
    })
    layout = { x = 0, y = 0, w = 18, h = 8 }
  }]
}
