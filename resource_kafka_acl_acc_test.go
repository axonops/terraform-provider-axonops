package main

import (
	"testing"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccKafkaACL_createAndRead(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_acl" "a" {
  cluster_name    = "kcluster"
  resource_type   = "TOPIC"
  resource_name   = "orders"
  principal       = "User:alice"
  operation       = "READ"
  permission_type = "ALLOW"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("axonops_kafka_acl.a", "resource_pattern_type", "LITERAL"),
					resource.TestCheckResourceAttr("axonops_kafka_acl.a", "host", "*"),
				),
			},
		},
	})
}

func TestAccKafkaACL_readDetectsExternalDelete(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_acl" "a" {
  cluster_name    = "kcluster"
  resource_type   = "TOPIC"
  resource_name   = "vanishing-acl"
  principal       = "User:bob"
  operation       = "WRITE"
  permission_type = "ALLOW"
}
`,
			},
			{
				PreConfig: func() {
					srv.deleteACLOutOfBand("kcluster", axonopsClient.KafkaACL{
						ResourceType:        "TOPIC",
						ResourceName:        "vanishing-acl",
						ResourcePatternType: "LITERAL",
						Principal:           "User:bob",
						Host:                "*",
						Operation:           "WRITE",
						PermissionType:      "ALLOW",
					})
				},
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_acl" "a" {
  cluster_name    = "kcluster"
  resource_type   = "TOPIC"
  resource_name   = "vanishing-acl"
  principal       = "User:bob"
  operation       = "WRITE"
  permission_type = "ALLOW"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_kafka_acl.a", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccKafkaACL_importRoundTrip(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_acl" "a" {
  cluster_name    = "kcluster"
  resource_type   = "TOPIC"
  resource_name   = "importable"
  principal       = "User:carol"
  operation       = "READ"
  permission_type = "ALLOW"
}
`,
			},
			{
				ResourceName:                         "axonops_kafka_acl.a",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "resource_name",
				ImportStateId:                        "kcluster/TOPIC/importable/LITERAL/User:carol/*/READ/ALLOW",
			},
		},
	})
}

func TestAccKafkaACLDataSource_listAndSingle(t *testing.T) {
	// Regression test: data sources used to receive a nil client because
	// provider.go never set resp.DataSourceData.
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_kafka_acl" "a" {
  cluster_name    = "kcluster"
  resource_type   = "TOPIC"
  resource_name   = "listed"
  principal       = "User:dave"
  operation       = "READ"
  permission_type = "ALLOW"
}

data "axonops_kafka_acl_list" "all" {
  cluster_name = "kcluster"
  depends_on   = [axonops_kafka_acl.a]
}

data "axonops_kafka_acl" "one" {
  cluster_name           = "kcluster"
  resource_type          = "TOPIC"
  resource_name          = "listed"
  resource_pattern_type  = "LITERAL"
  principal              = "User:dave"
  host                   = "*"
  operation              = "READ"
  permission_type        = "ALLOW"
  depends_on             = [axonops_kafka_acl.a]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_kafka_acl_list.all", "acls.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_kafka_acl.one", "principal", "User:dave"),
				),
			},
		},
	})
}
