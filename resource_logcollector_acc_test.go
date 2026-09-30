package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLogCollector_createReadImport(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_logcollector" "l" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "system-log"
  filename     = "/var/log/cassandra/system.log"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_logcollector.l", "uuid"),
					resource.TestCheckResourceAttr("axonops_logcollector.l", "date_format", "yyyy-MM-dd HH:mm:ss,SSS"),
				),
			},
			{
				ResourceName:                         "axonops_logcollector.l",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "filename",
				ImportStateId:                        "cassandra/ccluster//var/log/cassandra/system.log",
			},
		},
	})
}
