package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKafkaConnector_createAndUpdate(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_connect_connector" "c" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  name                 = "sink-connector"
  config = {
    "connector.class" = "io.confluent.connect.sink.SinkConnector"
    "tasks.max"        = "1"
  }
}
`,
				Check: resource.TestCheckResourceAttr("axonops_kafka_connect_connector.c", "config.tasks.max", "1"),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_connect_connector" "c" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  name                 = "sink-connector"
  config = {
    "connector.class" = "io.confluent.connect.sink.SinkConnector"
    "tasks.max"        = "2"
  }
}
`,
				Check: resource.TestCheckResourceAttr("axonops_kafka_connect_connector.c", "config.tasks.max", "2"),
			},
		},
	})
}

// TestAccKafkaConnector_serverInjectedConfigNoDiff verifies that config keys
// the Kafka Connect API injects server-side (the mock always adds "name" to
// the effective config map, mirroring real Kafka Connect) do not produce a
// perpetual diff, since refreshConnectorConfig only tracks keys already
// present in state.
func TestAccKafkaConnector_serverInjectedConfigNoDiff(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_connect_connector" "c" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  name                 = "injected"
  config = {
    "connector.class" = "io.confluent.connect.source.SourceConnector"
  }
}
`,
			},
			{
				// A plan-only refresh must show no changes even though the
				// mock server's config map for this connector now also
				// contains a server-injected "name" key.
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_connect_connector" "c" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  name                 = "injected"
  config = {
    "connector.class" = "io.confluent.connect.source.SourceConnector"
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccKafkaConnector_importRoundTrip(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_connect_connector" "c" {
  cluster_name         = "kcluster"
  connect_cluster_name = "connect1"
  name                 = "importable"
  config = {
    "connector.class" = "io.confluent.connect.sink.SinkConnector"
  }
}
`,
			},
			{
				ResourceName:                         "axonops_kafka_connect_connector.c",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"config"},
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateId:                        "kcluster/connect1/importable",
			},
		},
	})
}
