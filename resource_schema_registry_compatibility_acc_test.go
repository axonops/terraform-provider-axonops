package main

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccSchemaCompatibilityConfig(host, level string) string {
	return testAccProviderConfig(host) + fmt.Sprintf(`
resource "axonops_schema_registry_compatibility" "subject" {
  cluster_name        = "kcluster"
  subject             = "orders-value"
  compatibility_level = %[1]q
}

resource "axonops_schema_registry_compatibility" "global" {
  cluster_name        = "kcluster"
  compatibility_level = "FULL"
}
`, level)
}

func TestAccSchemaRegistryCompatibility_crud(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Set compatibility levels.
				Config: testAccSchemaCompatibilityConfig(srv.URL(), "BACKWARD"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("axonops_schema_registry_compatibility.subject", "id", "kcluster/orders-value"),
					resource.TestCheckResourceAttr("axonops_schema_registry_compatibility.global", "id", "kcluster"),
					testAccCheckMockCompatibility(srv, "orders-value", "BACKWARD"),
					testAccCheckMockCompatibility(srv, "", "FULL"),
				),
			},
			{
				// Update in place.
				Config: testAccSchemaCompatibilityConfig(srv.URL(), "FORWARD_TRANSITIVE"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_schema_registry_compatibility.subject", plancheck.ResourceActionUpdate),
					},
				},
				Check: testAccCheckMockCompatibility(srv, "orders-value", "FORWARD_TRANSITIVE"),
			},
			{
				// Drift: changed outside Terraform is set back.
				PreConfig: func() {
					srv.mu.Lock()
					srv.srCompat["kcluster"]["orders-value"] = "NONE"
					srv.mu.Unlock()
				},
				Config: testAccSchemaCompatibilityConfig(srv.URL(), "FORWARD_TRANSITIVE"),
				Check:  testAccCheckMockCompatibility(srv, "orders-value", "FORWARD_TRANSITIVE"),
			},
			{
				// Subject setting removed outside Terraform is recreated.
				PreConfig: func() { srv.deleteSchemaCompatibilityOutOfBand("kcluster", "orders-value") },
				Config:    testAccSchemaCompatibilityConfig(srv.URL(), "FORWARD_TRANSITIVE"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("axonops_schema_registry_compatibility.subject", plancheck.ResourceActionCreate),
					},
				},
			},
			{
				ResourceName:      "axonops_schema_registry_compatibility.subject",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "kcluster/orders-value",
			},
			{
				ResourceName:      "axonops_schema_registry_compatibility.global",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "kcluster",
			},
		},
		// The API cannot reset a level, so destroy leaves it in place.
		CheckDestroy: testAccCheckMockCompatibility(srv, "orders-value", "FORWARD_TRANSITIVE"),
	})
}

func testAccCheckMockCompatibility(srv *mockAxonOpsServer, subject, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got, _ := srv.schemaCompatibility("kcluster", subject)
		if got != want {
			return fmt.Errorf("compatibility of %q = %q, want %q", subject, got, want)
		}
		return nil
	}
}

func TestAccSchemaRegistryCompatibility_invalidInputs(t *testing.T) {
	srv := newAccTestServer(t)
	cfg := func(subject, level string) string {
		return testAccProviderConfig(srv.URL()) + fmt.Sprintf(`
resource "axonops_schema_registry_compatibility" "c" {
  cluster_name        = "kcluster"
  subject             = %q
  compatibility_level = %q
}
`, subject, level)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg("orders-value", "backward"), ExpectError: regexp.MustCompile(`value must be one of`)},
			{Config: cfg("orders-value", "STRICT"), ExpectError: regexp.MustCompile(`value must be one of`)},
			{Config: cfg("", "BACKWARD"), ExpectError: regexp.MustCompile(`at least 1`)},
		},
	})
}

func TestAccSchemaRegistryCompatibility_importInvalidID(t *testing.T) {
	srv := newAccTestServer(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_schema_registry_compatibility" "c" {
  cluster_name        = "kcluster"
  compatibility_level = "NONE"
}
`,
				ResourceName:  "axonops_schema_registry_compatibility.c",
				ImportState:   true,
				ImportStateId: "kcluster/",
				ExpectError:   regexp.MustCompile(`Invalid Import ID`),
			},
		},
	})
}
