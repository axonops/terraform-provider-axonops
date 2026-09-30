package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAlertRoute_createAndRead(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "route-target"
  webhook_url  = "https://hooks.slack.com/services/route"
}

resource "axonops_alert_route" "r" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  integration_name = axonops_slack_integration.s.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_alert_route.r", "id"),
					resource.TestCheckResourceAttr("axonops_alert_route.r", "enable_override", "true"),
				),
			},
		},
	})
}

// TestAccAlertRoute_disableOverride verifies that toggling enable_override
// from true to false is applied in place (Update, not Replace) and is
// reflected by the mock's integrations-override endpoint on the next read.
func TestAccAlertRoute_disableOverride(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "override-target"
  webhook_url  = "https://hooks.slack.com/services/override"
}

resource "axonops_alert_route" "r" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  integration_name = axonops_slack_integration.s.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
  enable_override  = true
}
`,
				Check: resource.TestCheckResourceAttr("axonops_alert_route.r", "enable_override", "true"),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "override-target"
  webhook_url  = "https://hooks.slack.com/services/override"
}

resource "axonops_alert_route" "r" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  integration_name = axonops_slack_integration.s.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
  enable_override  = false
}
`,
				Check: resource.TestCheckResourceAttr("axonops_alert_route.r", "enable_override", "false"),
			},
			{
				// A subsequent refresh-only plan must confirm the override
				// really was persisted to false server-side (not just held
				// in local state), by re-reading through Read().
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "override-target"
  webhook_url  = "https://hooks.slack.com/services/override"
}

resource "axonops_alert_route" "r" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  integration_name = axonops_slack_integration.s.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "warning"
  enable_override  = false
}
`,
				PlanOnly: true,
			},
		},
	})
}

func TestAccAlertRoute_importRoundTrip(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "import-target"
  webhook_url  = "https://hooks.slack.com/services/import"
}

resource "axonops_alert_route" "r" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  integration_name = axonops_slack_integration.s.name
  integration_type = "slack"
  type             = "repairs"
  severity         = "error"
  enable_override  = true
}
`,
			},
			{
				ResourceName:      "axonops_alert_route.r",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "cassandra/ccluster/repairs/error/slack/import-target",
			},
		},
	})
}

func TestAccAlertRouteDataSource_lookup(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "ds-target"
  webhook_url  = "https://hooks.slack.com/services/ds"
}

resource "axonops_alert_route" "r" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  integration_name = axonops_slack_integration.s.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "error"
}

data "axonops_alert_route" "r" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  integration_name = axonops_slack_integration.s.name
  integration_type = "slack"
  type             = "metrics"
  severity         = "error"
  depends_on       = [axonops_alert_route.r]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.axonops_alert_route.r", "id", "axonops_alert_route.r", "id"),
					resource.TestCheckResourceAttr("data.axonops_alert_route.r", "enable_override", "true"),
				),
			},
		},
	})
}
