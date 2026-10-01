package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSchema_createUpdateImport(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_schema" "s" {
  cluster_name = "kcluster"
  subject      = "orders-value"
  schema       = jsonencode({ type = "record", name = "Order", fields = [] })
  schema_type  = "AVRO"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("axonops_schema.s", "version", "1"),
					resource.TestCheckResourceAttrSet("axonops_schema.s", "schema_id"),
				),
			},
			{
				// Posting a new schema version bumps schema_id/version.
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_schema" "s" {
  cluster_name = "kcluster"
  subject      = "orders-value"
  schema       = jsonencode({ type = "record", name = "Order", fields = [{ name = "id", type = "string" }] })
  schema_type  = "AVRO"
}
`,
				Check: resource.TestCheckResourceAttr("axonops_schema.s", "version", "2"),
			},
			{
				ResourceName:                         "axonops_schema.s",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"schema"},
				ImportStateVerifyIdentifierAttribute: "subject",
				ImportStateId:                        "kcluster/orders-value",
			},
		},
	})
}
