package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCassandraScheduledRepair_createReadImport(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_scheduled_repair" "r" {
  cluster_name  = "ccluster"
  tag           = "weekly-repair"
  keyspace      = "myks"
  schedule_expr = "0 3 * * 0"
}
`,
				Check: resource.TestCheckResourceAttrSet("axonops_cassandra_scheduled_repair.r", "repair_id"),
			},
			{
				ResourceName:                         "axonops_cassandra_scheduled_repair.r",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "tag",
				ImportStateId:                        "ccluster/weekly-repair",
			},
		},
	})
}
