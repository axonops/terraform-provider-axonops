package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLogAlertRule_createReadImport(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_log_alert_rule" "l" {
  cluster_name   = "ccluster"
  cluster_type   = "cassandra"
  name           = "node-down"
  content        = "+is +now +DOWN"
  warning_value  = 1
  critical_value = 5
  duration       = "5m"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_log_alert_rule.l", "id"),
					resource.TestCheckResourceAttr("axonops_log_alert_rule.l", "present", "true"),
				),
			},
			{
				ResourceName:      "axonops_log_alert_rule.l",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccAlertRuleImportStateID("axonops_log_alert_rule.l"),
			},
		},
	})
}
