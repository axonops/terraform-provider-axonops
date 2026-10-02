package main

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccApiTokenConfig(host, name, rotation string) string {
	return testAccProviderConfig(host) + fmt.Sprintf(`
resource "axonops_api_token" "t" {
  name          = %q
  allowed_roles = ["testorg/kafka/kcluster/readonly", "testorg/cassandra/admin"]
  expires_at    = "2030-01-01T00:00:00Z"
  rotation_triggers = {
    rotated = %q
  }

  lifecycle {
    create_before_destroy = true
  }
}
`, name, rotation)
}

func TestAccApiToken_crud(t *testing.T) {
	srv := newAccTestServer(t)
	var firstID, firstSecret string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create.
				Config: testAccApiTokenConfig(srv.URL(), "ci", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_api_token.t", "id"),
					resource.TestCheckResourceAttrSet("axonops_api_token.t", "secret"),
					resource.TestCheckResourceAttrSet("axonops_api_token.t", "created_at"),
					resource.TestCheckResourceAttr("axonops_api_token.t", "allowed_roles.#", "2"),
					resource.TestCheckResourceAttr("axonops_api_token.t", "expires_at", "2030-01-01T00:00:00Z"),
					func(s *terraform.State) error {
						attrs := s.RootModule().Resources["axonops_api_token.t"].Primary.Attributes
						firstID, firstSecret = attrs["id"], attrs["secret"]
						return nil
					},
				),
			},
			{
				// Renaming is state-only and keeps the token.
				Config: testAccApiTokenConfig(srv.URL(), "ci-renamed", "1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_api_token.t", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("axonops_api_token.t", "name", "ci-renamed"),
					resource.TestCheckResourceAttrPtr("axonops_api_token.t", "id", &firstID),
					resource.TestCheckResourceAttrPtr("axonops_api_token.t", "secret", &firstSecret),
				),
			},
			{
				// Rotate: a new token is created and the old one revoked.
				Config: testAccApiTokenConfig(srv.URL(), "ci-renamed", "2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_api_token.t", plancheck.ResourceActionReplace),
					},
				},
				Check: func(s *terraform.State) error {
					attrs := s.RootModule().Resources["axonops_api_token.t"].Primary.Attributes
					if attrs["id"] == firstID || attrs["secret"] == firstSecret {
						return fmt.Errorf("token was not rotated")
					}
					if srv.apiTokenExists(firstID) {
						return fmt.Errorf("old token %s was not revoked", firstID)
					}
					if n := srv.apiTokenCount(); n != 1 {
						return fmt.Errorf("expected 1 token, got %d", n)
					}
					return nil
				},
			},
			{
				// Import: the secret cannot be recovered.
				ResourceName:            "axonops_api_token.t",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name", "secret", "rotation_triggers"},
			},
		},
		// Revoke.
		CheckDestroy: func(s *terraform.State) error {
			if n := srv.apiTokenCount(); n != 0 {
				return fmt.Errorf("expected all tokens revoked, %d left", n)
			}
			return nil
		},
	})
}

func TestAccApiToken_revokedOutOfBand(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccApiTokenConfig(srv.URL(), "ci", "1")},
			{
				PreConfig: func() {
					srv.mu.Lock()
					srv.apiTokens = nil
					srv.mu.Unlock()
				},
				Config: testAccApiTokenConfig(srv.URL(), "ci", "1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_api_token.t", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestAccApiToken_invalidInputs(t *testing.T) {
	srv := newAccTestServer(t)
	cfg := func(roles, expires string) string {
		return testAccProviderConfig(srv.URL()) + fmt.Sprintf(`
resource "axonops_api_token" "t" {
  name          = "bad"
  allowed_roles = %s
  expires_at    = %q
}
`, roles, expires)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg(`["testorg/owner"]`, "2030-01-01T00:00:00Z"), ExpectError: regexp.MustCompile(`must be <org>/<role>`)},
			{Config: cfg(`["readonly"]`, "2030-01-01T00:00:00Z"), ExpectError: regexp.MustCompile(`must be <org>/<role>`)},
			{Config: cfg(`["a/b/c/d/readonly"]`, "2030-01-01T00:00:00Z"), ExpectError: regexp.MustCompile(`must be <org>/<role>`)},
			{Config: cfg(`[]`, "2030-01-01T00:00:00Z"), ExpectError: regexp.MustCompile(`at least 1`)},
			{Config: cfg(`["testorg/admin", "testorg/admin"]`, "2030-01-01T00:00:00Z"), ExpectError: regexp.MustCompile(`duplicate values`)},
			{Config: cfg(`["testorg/admin"]`, "next year"), ExpectError: regexp.MustCompile(`RFC 3339`)},
			{Config: cfg(`["testorg/admin"]`, "2100-01-01T00:00:00Z"), ExpectError: regexp.MustCompile(`out of range`)},
		},
	})
}
