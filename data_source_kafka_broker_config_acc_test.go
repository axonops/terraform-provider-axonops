package main

import (
	"regexp"
	"testing"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKafkaBrokerConfig_read(t *testing.T) {
	srv := newAccTestServer(t)
	retention, threads := "168", "8"
	srv.seedBroker("kcluster", axonopsClient.KafkaBrokerInfo{
		BrokerID: 1,
		Address:  "broker-1:9092",
		Configs: []axonopsClient.BrokerConfigEntry{
			{Name: "log.retention.hours", Value: &retention, Source: "STATIC_BROKER_CONFIG", IsExplicitlySet: true},
			{Name: "num.io.threads", Value: &threads, Source: "DEFAULT_CONFIG", IsDefaultValue: true},
			{Name: "ssl.keystore.password", Source: "STATIC_BROKER_CONFIG", IsSensitive: true},
		},
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// All configs.
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_kafka_broker_config" "b" {
  cluster_name = "kcluster"
  broker_id    = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "address", "broker-1:9092"),
					resource.TestCheckNoResourceAttr("data.axonops_kafka_broker_config.b", "rack"),
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "configs.#", "3"),
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "values.log.retention.hours", "168"),
					// Sensitive configs have no value and are left out of values.
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "values.%", "2"),
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "configs.2.is_sensitive", "true"),
					resource.TestCheckNoResourceAttr("data.axonops_kafka_broker_config.b", "configs.2.value"),
				),
			},
			{
				// Filtered by name.
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_kafka_broker_config" "b" {
  cluster_name = "kcluster"
  broker_id    = 1
  config_names = ["num.io.threads"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "configs.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "configs.0.is_default", "true"),
					resource.TestCheckResourceAttr("data.axonops_kafka_broker_config.b", "values.num.io.threads", "8"),
				),
			},
		},
	})
}

func TestAccKafkaBrokerConfig_errors(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Unknown broker.
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_kafka_broker_config" "b" {
  cluster_name = "kcluster"
  broker_id    = 42
}
`,
				ExpectError: regexp.MustCompile(`Unable to read Kafka broker config`),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_kafka_broker_config" "b" {
  cluster_name = "kcluster"
  broker_id    = -1
}
`,
				ExpectError: regexp.MustCompile(`at least 0`),
			},
		},
	})
}
