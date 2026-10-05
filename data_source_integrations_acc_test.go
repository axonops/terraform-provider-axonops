package main

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceIntegrations_listAndFilterByType(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "alerts-channel"
  webhook_url  = "https://hooks.slack.com/services/secret"
  channel      = "#alerts"
}

resource "axonops_pagerduty_integration" "p" {
  cluster_name    = "ccluster"
  cluster_type    = "cassandra"
  name            = "oncall"
  integration_key = "secret-key"
}

data "axonops_integrations" "all" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  depends_on   = [axonops_slack_integration.s, axonops_pagerduty_integration.p]
}

data "axonops_integrations" "slack" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  type         = "SLACK"
  depends_on   = [axonops_slack_integration.s, axonops_pagerduty_integration.p]
}

data "axonops_integrations" "none" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  type         = "opsgenie"
  depends_on   = [axonops_slack_integration.s, axonops_pagerduty_integration.p]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_integrations.all", "integrations.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_integrations.all", "integrations.0.type", "pagerduty"),
					resource.TestCheckResourceAttr("data.axonops_integrations.all", "integrations.0.name", "oncall"),
					resource.TestCheckResourceAttrPair("data.axonops_integrations.all", "integrations.0.id", "axonops_pagerduty_integration.p", "id"),
					resource.TestCheckResourceAttr("data.axonops_integrations.all", "integrations.1.type", "slack"),
					resource.TestCheckResourceAttr("data.axonops_integrations.slack", "integrations.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_integrations.slack", "integrations.0.name", "alerts-channel"),
					resource.TestCheckResourceAttrPair("data.axonops_integrations.slack", "integrations.0.id", "axonops_slack_integration.s", "id"),
					resource.TestCheckResourceAttr("data.axonops_integrations.none", "integrations.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceIntegrations_invalidClusterType(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_integrations" "bad" {
  cluster_name = "ccluster"
  cluster_type = "postgres"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}
