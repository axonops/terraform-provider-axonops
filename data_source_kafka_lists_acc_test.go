package main

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceKafkaTopics_list(t *testing.T) {
	srv := newAccTestServer(t)

	config := testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "b" {
  name               = "payments"
  cluster_name       = "kcluster"
  partitions         = 6
  replication_factor = 3
}

resource "axonops_kafka_topic" "a" {
  name               = "orders"
  cluster_name       = "kcluster"
  partitions         = 3
  replication_factor = 2
}

data "axonops_kafka_topics" "all" {
  cluster_name = "kcluster"
  depends_on   = [axonops_kafka_topic.a, axonops_kafka_topic.b]
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "names.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "names.0", "orders"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "names.1", "payments"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "topics.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "topics.0.name", "orders"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "topics.0.partitions", "3"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "topics.0.replication_factor", "2"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "topics.1.name", "payments"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.all", "topics.1.partitions", "6"),
				),
			},
		},
	})
}

func TestAccDataSourceKafkaTopics_emptyCluster(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_kafka_topics" "none" {
  cluster_name = "empty"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.none", "names.#", "0"),
					resource.TestCheckResourceAttr("data.axonops_kafka_topics.none", "topics.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceKafkaConnectors_listAndFilter(t *testing.T) {
	srv := newAccTestServer(t)

	resources := testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_connect_connector" "sink" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  name                 = "s3-sink"
  config = {
    "connector.class" = "io.confluent.connect.s3.S3SinkConnector"
    "tasks.max"       = "2"
  }
}

resource "axonops_kafka_connect_connector" "source" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  name                 = "jdbc-source"
  config = {
    "connector.class" = "io.confluent.connect.jdbc.JdbcSourceConnector"
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: resources + `
data "axonops_kafka_connectors" "all" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  depends_on           = [axonops_kafka_connect_connector.sink, axonops_kafka_connect_connector.source]
}

data "axonops_kafka_connectors" "sinks" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  type                 = "sink"
  depends_on           = [axonops_kafka_connect_connector.sink, axonops_kafka_connect_connector.source]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "names.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "names.0", "jdbc-source"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "names.1", "s3-sink"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "connectors.0.type", "source"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "connectors.0.tasks_max", ""),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "connectors.1.class", "io.confluent.connect.s3.S3SinkConnector"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "connectors.1.state", "RUNNING"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.all", "connectors.1.tasks_max", "2"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.sinks", "names.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_kafka_connectors.sinks", "names.0", "s3-sink"),
				),
			},
		},
	})
}

func TestAccDataSourceKafkaConnectors_invalidType(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_kafka_connectors" "bad" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  type                 = "transform"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}
