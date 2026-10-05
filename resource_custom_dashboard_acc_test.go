package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	axonopsClient "terraform-provider-axonops/client"

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
	var firstPanelUUID, emptyRowUUID string

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
						if err := checkUIRenderable(dash); err != nil {
							return err
						}
						emptyRowUUID = dash.Panels[0].UUID
						// The server groups panels under the preceding row.
						if dash.Panels[2].Group != dash.Panels[1].UUID {
							return fmt.Errorf("panel group = %q, want row uuid %q", dash.Panels[2].Group, dash.Panels[1].UUID)
						}
						// Configured panels sit one grid row below the hidden row.
						if dash.Panels[1].Layout.Y != 1 || dash.Panels[2].Layout.Y != 2 {
							return fmt.Errorf("stored y = %d, %d; want 1, 2", dash.Panels[1].Layout.Y, dash.Panels[2].Layout.Y)
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
						attrs := s.RootModule().Resources["axonops_custom_dashboard.d"].Primary.Attributes
						if got := attrs["panels.0.uuid"]; got != firstPanelUUID {
							return fmt.Errorf("panel uuid changed from %s to %s", firstPanelUUID, got)
						}
						dash := srv.dashboardV2("kafka", "kcluster", attrs["id"])
						if err := checkUIRenderable(dash); err != nil {
							return err
						}
						if dash.Panels[0].UUID != emptyRowUUID {
							return fmt.Errorf("hidden row uuid changed from %s to %s", emptyRowUUID, dash.Panels[0].UUID)
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

	resource.Test(t, resource.TestCase{
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
  name         = "Reserved"
  panels       = [{ title = "__EMPTY_ROW__", type = "row", layout = { x = 0, y = 0, w = 18, h = 1 } }]
}
`,
				ExpectError: regexp.MustCompile(`value must be none of`),
			},
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

// checkUIRenderable checks what the AxonOps UI needs to render a dashboard:
// the hidden __EMPTY_ROW__ row first, and details on every panel.
func checkUIRenderable(dash *axonopsClient.CustomDashboard) error {
	if dash == nil {
		return fmt.Errorf("dashboard not stored")
	}
	if len(dash.Panels) == 0 || dash.Panels[0].Type != "row" || dash.Panels[0].Title != emptyRowTitle {
		return fmt.Errorf("first panel is not the hidden %s row", emptyRowTitle)
	}
	for _, p := range dash.Panels {
		if len(p.Details) == 0 || string(p.Details) == "null" {
			return fmt.Errorf("panel %q has no details", p.Title)
		}
	}
	return nil
}

func TestAccCustomDashboard_importUIDashboard(t *testing.T) {
	srv := newAccTestServer(t)
	// A dashboard as the AxonOps UI saves it: hidden row first, panels below.
	srv.mu.Lock()
	srv.dashboardsV2["cassandra/ccluster"] = &axonopsClient.DashboardTemplate{
		Type: "cassandra",
		Dashboards: []axonopsClient.CustomDashboard{{
			UUID: "ui-dash",
			Name: "From UI",
			Panels: []axonopsClient.CustomPanel{
				{UUID: "ui-empty", Type: "row", Title: emptyRowTitle, Details: json.RawMessage(emptyRowDetails),
					Layout: axonopsClient.PanelLayout{W: 18, H: 1, I: "ui-empty"}},
				{UUID: "ui-cpu", Type: "line-chart", Title: "CPU", Group: "ui-empty",
					Details: json.RawMessage(`{"queries":[{"query":"host_CPU_Percent_Merge","legend":"{{host_id}}"}]}`),
					Layout:  axonopsClient.PanelLayout{W: 18, H: 6, Y: 1, I: "ui-cpu"}},
			},
		}},
	}
	srv.mu.Unlock()

	config := testAccProviderConfig(srv.URL()) + `
resource "axonops_custom_dashboard" "ui" {
  cluster_type = "cassandra"
  cluster_name = "ccluster"
  name         = "From UI"
  panels = [{
    title   = "CPU"
    type    = "line-chart"
    details = jsonencode({ queries = [{ query = "host_CPU_Percent_Merge", legend = "{{host_id}}" }] })
    layout  = { x = 0, y = 0, w = 18, h = 6 }
  }]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Adopting the UI dashboard hides its __EMPTY_ROW__ and shows no diff.
				Config:             config,
				ResourceName:       "axonops_custom_dashboard.ui",
				ImportState:        true,
				ImportStateId:      "cassandra/ccluster/ui-dash",
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					a := states[0].Attributes
					if a["panels.#"] != "1" || a["panels.0.uuid"] != "ui-cpu" || a["panels.0.layout.y"] != "0" {
						return fmt.Errorf("unexpected import: panels.#=%s uuid=%s y=%s", a["panels.#"], a["panels.0.uuid"], a["panels.0.layout.y"])
					}
					return nil
				},
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
