package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccSilence_createAndRead(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_silence" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  duration     = "1h"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("axonops_silence.s", "active", "true"),
					resource.TestCheckResourceAttr("axonops_silence.s", "cron_expr", "0 * * * *"),
					resource.TestCheckResourceAttrSet("axonops_silence.s", "id"),
				),
			},
		},
	})
}

// TestAccSilence_multipleWithDefaultCronDoNotCollide exercises
// findNewSilenceID's disambiguation logic: several silences created with the
// same (default) cron_expr and duration must each end up with their own
// distinct server-assigned ID rather than all resolving to the same one.
func TestAccSilence_multipleWithDefaultCronDoNotCollide(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_silence" "s1" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  duration     = "1h"
}

resource "axonops_silence" "s2" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  duration     = "1h"
}

resource "axonops_silence" "s3" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  duration     = "1h"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_silence.s1", "id"),
					resource.TestCheckResourceAttrSet("axonops_silence.s2", "id"),
					resource.TestCheckResourceAttrSet("axonops_silence.s3", "id"),
					testAccCheckSilenceIDsDistinct("axonops_silence.s1", "axonops_silence.s2", "axonops_silence.s3"),
				),
			},
		},
	})
}

func TestAccSilence_importRoundTrip(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_silence" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  duration     = "30m"
  note         = "maintenance"
}
`,
			},
			{
				ResourceName:      "axonops_silence.s",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccSilenceImportStateID("axonops_silence.s"),
			},
		},
	})
}

// testAccSilenceImportStateID builds the cluster_type/cluster_name/id import
// ID from the resource's own state, since the silence ID is assigned
// server-side and not known ahead of time.
func testAccSilenceImportStateID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceName)
		}
		attrs := rs.Primary.Attributes
		return fmt.Sprintf("%s/%s/%s", attrs["cluster_type"], attrs["cluster_name"], attrs["id"]), nil
	}
}

// testAccCheckSilenceIDsDistinct asserts that every named resource resolved
// to a different "id" value, catching regressions where findNewSilenceID
// fails to disambiguate concurrently created silences that share identical
// user-supplied fields.
func testAccCheckSilenceIDsDistinct(resourceNames ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		seen := map[string]string{}
		for _, name := range resourceNames {
			rs, ok := s.RootModule().Resources[name]
			if !ok {
				return fmt.Errorf("resource not found in state: %s", name)
			}
			id := rs.Primary.Attributes["id"]
			if id == "" {
				return fmt.Errorf("resource %s has empty id", name)
			}
			if other, dup := seen[id]; dup {
				return fmt.Errorf("resources %s and %s both resolved to id %q; expected distinct IDs", other, name, id)
			}
			seen[id] = name
		}
		return nil
	}
}
