package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSlackIntegration_createAndUpdate(t *testing.T) {
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
  webhook_url  = "https://hooks.slack.com/services/original"
  channel      = "#alerts"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_slack_integration.s", "id"),
					resource.TestCheckResourceAttr("axonops_slack_integration.s", "channel", "#alerts"),
				),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "alerts-channel"
  webhook_url  = "https://hooks.slack.com/services/updated"
  channel      = "#ops"
}
`,
				Check: resource.TestCheckResourceAttr("axonops_slack_integration.s", "channel", "#ops"),
			},
		},
	})
}

// TestAccSlackIntegration_importDoesNotStoreMaskedSecret verifies that
// webhook_url (Sensitive, Required) is never populated with the masked value
// the mock API returns on GET. ImportState for this resource intentionally
// leaves webhook_url unset (see integrationImportSecretWarning in
// resource_slack_integration.go), so it must be absent from the imported
// state rather than equal to the server's masked placeholder.
func TestAccSlackIntegration_importDoesNotStoreMaskedSecret(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_slack_integration" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "masked-check"
  webhook_url  = "https://hooks.slack.com/services/real-secret"
}
`,
			},
			{
				ResourceName:            "axonops_slack_integration.s",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook_url"},
				ImportStateId:           "cassandra/ccluster/masked-check",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected exactly 1 imported instance, got %d", len(states))
					}
					if v, ok := states[0].Attributes["webhook_url"]; ok && v == maskedSecretValue {
						return fmt.Errorf("webhook_url was populated with the server's masked placeholder %q; ImportState must leave it unset", maskedSecretValue)
					}
					return nil
				},
			},
		},
	})
}
