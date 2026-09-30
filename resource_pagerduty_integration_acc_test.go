package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPagerDutyIntegration_createUpdateImport(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_pagerduty_integration" "p" {
  cluster_name    = "ccluster"
  cluster_type    = "cassandra"
  name            = "oncall"
  integration_key = "original-key"
}
`,
				Check: resource.TestCheckResourceAttrSet("axonops_pagerduty_integration.p", "id"),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_pagerduty_integration" "p" {
  cluster_name    = "ccluster"
  cluster_type    = "cassandra"
  name            = "oncall"
  integration_key = "rotated-key"
}
`,
			},
			{
				ResourceName:            "axonops_pagerduty_integration.p",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"integration_key"},
				ImportStateId:           "cassandra/ccluster/oncall",
			},
		},
	})
}
