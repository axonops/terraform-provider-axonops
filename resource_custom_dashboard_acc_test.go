package main

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccCustomDashboardConfig(host, name, query string) string {
	return testAccProviderConfig(host) + fmt.Sprintf(`
resource "axonops_custom_dashboard" "d" {
  cluster_type = "kafka"
  cluster_name = "kcluster"
  name         = %q

  filters = [{
    name   = "host"
    label  = "Host"
    type   = "custom"
    multi  = true
    values = "a,b"
  }]

  panels = [
    {
      title  = "Brokers"
      type   = "row"
      layout = { x = 0, y = 0, w = 18, h = 1 }
    },
    {
      title = "Messages in"
      type  = "line-chart"
      details = jsonencode({
        queries    = [{ query = %q, legend = "{{host}}" }]
        nullAsZero = true
      })
      layout = { x = 0, y = 1, w = 9, h = 6 }
    },
  ]
}
`, name, query)
}

func TestAccCustomDashboard_create(t *testing.T) {
	srv := newAccTestServer(t)
	var firstPanelUUID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomDashboardConfig(srv.URL(), "Kafka traffic", "kafka_messages_in"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_custom_dashboard.d", "id"),
					resource.TestCheckResourceAttr("axonops_custom_dashboard.d", "panels.#", "2"),
					resource.TestCheckResourceAttrSet("axonops_custom_dashboard.d", "panels.0.uuid"),
					resource.TestCheckResourceAttr("axonops_custom_dashboard.d", "panels.1.layout.w", "9"),
					resource.TestCheckResourceAttr("axonops_custom_dashboard.d", "filters.0.values", "a,b"),
					func(s *terraform.State) error {
						attrs := s.RootModule().Resources["axonops_custom_dashboard.d"].Primary.Attributes
						firstPanelUUID = attrs["panels.0.uuid"]
						dash := srv.dashboardV2("kafka", "kcluster", attrs["id"])
						if dash == nil {
							return fmt.Errorf("dashboard %s not stored", attrs["id"])
						}
						// The server groups panels under the preceding row.
						if dash.Panels[1].Group != dash.Panels[0].UUID {
							return fmt.Errorf("panel group = %q, want row uuid %q", dash.Panels[1].Group, dash.Panels[0].UUID)
						}
						// Dashboards not managed by Terraform are kept.
						if srv.dashboardV2("kafka", "kcluster", builtinDashboardUUID) == nil {
							return fmt.Errorf("built-in dashboard was dropped")
						}
						return nil
					},
				),
			},
			{
				// Renaming and changing a query updates in place and keeps panel UUIDs.
				Config: testAccCustomDashboardConfig(srv.URL(), "Kafka throughput", "kafka_bytes_in"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_custom_dashboard.d", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("axonops_custom_dashboard.d", "name", "Kafka throughput"),
					func(s *terraform.State) error {
						got := s.RootModule().Resources["axonops_custom_dashboard.d"].Primary.Attributes["panels.0.uuid"]
						if got != firstPanelUUID {
							return fmt.Errorf("panel uuid changed from %s to %s", firstPanelUUID, got)
						}
						return nil
					},
				),
			},
			{
				ResourceName:      "axonops_custom_dashboard.d",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return "kafka/kcluster/" + s.RootModule().Resources["axonops_custom_dashboard.d"].Primary.ID, nil
				},
			},
			{
				// Dashboard deleted in the UI: Terraform recreates it.
				PreConfig: func() {
					for _, d := range srv.dashboardsV2List("kafka", "kcluster") {
						if d.Name == "Kafka throughput" {
							srv.deleteDashboardOutOfBand("kafka", "kcluster", d.UUID)
						}
					}
				},
				Config: testAccCustomDashboardConfig(srv.URL(), "Kafka throughput", "kafka_bytes_in"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_custom_dashboard.d", plancheck.ResourceActionCreate),
					},
				},
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			for _, d := range srv.dashboardsV2List("kafka", "kcluster") {
				if d.UUID != builtinDashboardUUID {
					return fmt.Errorf("dashboard %s still exists after destroy", d.UUID)
				}
			}
			return nil
		},
	})
}

func TestAccCustomDashboard_invalidInputs(t *testing.T) {
	srv := newAccTestServer(t)
	base := func(clusterType, details string, w int) string {
		return testAccProviderConfig(srv.URL()) + fmt.Sprintf(`
resource "axonops_custom_dashboard" "d" {
  cluster_type = %q
  cluster_name = "kcluster"
  name         = "Bad"
  panels = [{
    title   = "p"
    type    = "line-chart"
    details = %q
    layout  = { x = 0, y = 0, w = %d, h = 1 }
  }]
}
`, clusterType, details, w)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: base("kafka", "not json", 1), ExpectError: regexp.MustCompile(`Invalid JSON`)},
			{Config: base("kafka", `["array"]`, 1), ExpectError: regexp.MustCompile(`Invalid JSON`)},
			{Config: base("kafka", "{}", 0), ExpectError: regexp.MustCompile(`must be at least 1`)},
			{Config: base("redis", "{}", 1), ExpectError: regexp.MustCompile(`cluster_type`)},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_custom_dashboard" "d" {
  cluster_type = "kafka"
  cluster_name = "kcluster"
  name         = "Empty"
  panels       = []
}
`,
				ExpectError: regexp.MustCompile(`at least 1`),
			},
		},
	})
}
