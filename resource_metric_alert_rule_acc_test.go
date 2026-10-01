package main

import (
	"regexp"
	"testing"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// seedMetricDashboard populates the mock server with a dashboard/panel pair
// that resource_metric_alert_rule.go's dashboard-resolution logic can find,
// mirroring the shape of client.DashboardTemplateResponse.
func seedMetricDashboardFixture(srv *mockAxonOpsServer, clusterType, clusterName string) {
	srv.seedDashboard(clusterType, clusterName, axonopsClient.Dashboard{
		UUID: "dash-uuid-1",
		Name: "Overview",
		Panels: []axonopsClient.DashboardPanel{
			{
				UUID:  "panel-uuid-1",
				Title: "CPU Usage",
				Type:  "timeseries",
				Details: axonopsClient.DashboardPanelDetails{
					Queries: []axonopsClient.DashboardPanelQuery{
						{Query: `cpu_usage{dc=~'$dc'}`},
					},
				},
			},
		},
	})
}

func TestAccMetricAlertRule_createAndRead(t *testing.T) {
	srv := newAccTestServer(t)
	seedMetricDashboardFixture(srv, "cassandra", "ccluster")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_metric_alert_rule" "m" {
  cluster_name   = "ccluster"
  cluster_type   = "cassandra"
  name           = "cpu-high"
  operator       = ">"
  warning_value  = 70
  critical_value = 90
  duration       = "5m"
  dashboard      = "Overview"
  chart          = "CPU Usage"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_metric_alert_rule.m", "id"),
					resource.TestCheckResourceAttr("axonops_metric_alert_rule.m", "correlation_id", "panel-uuid-1"),
				),
			},
		},
	})
}

func TestAccMetricAlertRule_unresolvableDashboardFailsAtApply(t *testing.T) {
	srv := newAccTestServer(t)
	// No dashboard fixture seeded.

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_metric_alert_rule" "m" {
  cluster_name   = "ccluster"
  cluster_type   = "cassandra"
  name           = "cpu-high"
  operator       = ">"
  warning_value  = 70
  critical_value = 90
  duration       = "5m"
  dashboard      = "Nonexistent"
  chart          = "Missing Chart"
}
`,
				ExpectError: regexp.MustCompile(`dashboard "Nonexistent" not found`),
			},
		},
	})
}

// TestAccMetricAlertRule_importFailsWhenChartUnresolvable verifies that
// importing an alert rule whose correlation_id does not match any panel UUID
// in the cluster's dashboard templates fails the import with a clear error,
// since dashboard/chart are Required attributes the import cannot leave
// unset.
func TestAccMetricAlertRule_importFailsWhenChartUnresolvable(t *testing.T) {
	srv := newAccTestServer(t)
	seedMetricDashboardFixture(srv, "cassandra", "ccluster")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_metric_alert_rule" "m" {
  cluster_name   = "ccluster"
  cluster_type   = "cassandra"
  name           = "cpu-high"
  operator       = ">"
  warning_value  = 70
  critical_value = 90
  duration       = "5m"
  dashboard      = "Overview"
  chart          = "CPU Usage"
}
`,
			},
			{
				PreConfig: func() {
					// Simulate the dashboard/panel being renamed or removed
					// out-of-band: the alert rule's stored correlationId no
					// longer resolves to any panel.
					srv.mu.Lock()
					srv.dashboards[clusterKey("cassandra", "ccluster")] = &axonopsClient.DashboardTemplateResponse{}
					srv.mu.Unlock()
				},
				ResourceName:      "axonops_metric_alert_rule.m",
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateIdFunc: testAccAlertRuleImportStateID("axonops_metric_alert_rule.m"),
				ExpectError:       regexp.MustCompile(`Could not resolve dashboard/chart names from correlation ID`),
			},
		},
	})
}

// TestAccMetricAlertRule_escapedDollarChartNoDrift covers a chart title with a
// literal "$" configured as "$$": the post-apply plan must keep the
// configured spelling instead of showing a diff to the API's "$" title.
func TestAccMetricAlertRule_escapedDollarChartNoDrift(t *testing.T) {
	srv := newAccTestServer(t)
	srv.seedDashboard("cassandra", "ccluster", axonopsClient.Dashboard{
		UUID: "dash-uuid-1",
		Name: "Overview",
		Panels: []axonopsClient.DashboardPanel{{
			UUID:  "panel-uuid-1",
			Title: "Max Size per $groupBy",
			Type:  "timeseries",
			Details: axonopsClient.DashboardPanelDetails{
				Queries: []axonopsClient.DashboardPanelQuery{{Query: `max(cas_size) by ($groupBy)`}},
			},
		}},
	})
	config := testAccProviderConfig(srv.URL()) + `
resource "axonops_metric_alert_rule" "m" {
  cluster_name   = "ccluster"
  cluster_type   = "cassandra"
  name           = "size-high"
  operator       = ">"
  warning_value  = 70
  critical_value = 90
  duration       = "5m"
  dashboard      = "Overview"
  chart          = "Max Size per $$groupBy"
  metric         = "max(cas_size) by (keyspace)"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr("axonops_metric_alert_rule.m", "chart", "Max Size per $$groupBy"),
			},
			{Config: config, PlanOnly: true},
		},
	})
}
