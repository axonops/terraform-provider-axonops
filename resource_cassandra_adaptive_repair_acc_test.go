package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCassandraAdaptiveRepair_createUpdateImport(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_adaptive_repair" "a" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  active       = true
}
`,
				Check: resource.TestCheckResourceAttr("axonops_cassandra_adaptive_repair.a", "active", "true"),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_adaptive_repair" "a" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  active       = false
}
`,
				Check: resource.TestCheckResourceAttr("axonops_cassandra_adaptive_repair.a", "active", "false"),
			},
			{
				ResourceName:                         "axonops_cassandra_adaptive_repair.a",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "cluster_name",
				ImportStateId:                        "cassandra/ccluster",
			},
		},
	})
}
