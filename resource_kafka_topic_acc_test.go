package main

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccKafkaTopic_createAndUpdatePartitions(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "orders"
  cluster_name        = "kcluster"
  partitions          = 3
  replication_factor  = 2
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("axonops_kafka_topic.t", "name", "orders"),
					resource.TestCheckResourceAttr("axonops_kafka_topic.t", "partitions", "3"),
					resource.TestCheckResourceAttr("axonops_kafka_topic.t", "replication_factor", "2"),
				),
			},
		},
	})
}

func TestAccKafkaTopic_increasePartitions(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "events"
  cluster_name        = "kcluster"
  partitions          = 3
  replication_factor  = 1
}
`,
				Check: resource.TestCheckResourceAttr("axonops_kafka_topic.t", "partitions", "3"),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "events"
  cluster_name        = "kcluster"
  partitions          = 6
  replication_factor  = 1
}
`,
				Check: resource.TestCheckResourceAttr("axonops_kafka_topic.t", "partitions", "6"),
			},
		},
	})
}

func TestAccKafkaTopic_decreasePartitionsFailsAtPlan(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "shrink"
  cluster_name        = "kcluster"
  partitions          = 6
  replication_factor  = 1
}
`,
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "shrink"
  cluster_name        = "kcluster"
  partitions          = 3
  replication_factor  = 1
}
`,
				ExpectError: regexp.MustCompile("Cannot Decrease Partitions"),
			},
		},
	})
}

func TestAccKafkaTopic_renameRequiresReplace(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "original-name"
  cluster_name        = "kcluster"
  partitions          = 1
  replication_factor  = 1
}
`,
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "renamed"
  cluster_name        = "kcluster"
  partitions          = 1
  replication_factor  = 1
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_kafka_topic.t", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestAccKafkaTopic_readDetectsExternalDelete(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "vanishing"
  cluster_name        = "kcluster"
  partitions          = 1
  replication_factor  = 1
}
`,
			},
			{
				PreConfig: func() {
					srv.deleteTopicOutOfBand("kcluster", "vanishing")
				},
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "vanishing"
  cluster_name        = "kcluster"
  partitions          = 1
  replication_factor  = 1
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_kafka_topic.t", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccKafkaTopic_importRoundTrip(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_topic" "t" {
  name                = "importme"
  cluster_name        = "kcluster"
  partitions          = 2
  replication_factor  = 1
  config = {
    retention_ms = "3600000"
  }
}
`,
			},
			{
				ResourceName:                         "axonops_kafka_topic.t",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        "kcluster/importme",
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}
