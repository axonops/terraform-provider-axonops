package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCassandraBackup_createReadImport(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_backup" "b" {
  cluster_name    = "ccluster"
  cluster_type    = "cassandra"
  tag             = "nightly"
  local_retention = "7d"
  datacenters     = ["dc1"]
}
`,
				Check: resource.TestCheckResourceAttrSet("axonops_cassandra_backup.b", "id"),
			},
			{
				ResourceName:                         "axonops_cassandra_backup.b",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "tag",
				ImportStateId:                        "cassandra/ccluster/nightly",
				// remote_type/remote_path (Optional, no Computed default) and
				// remote_retention round-trip as empty string rather than the
				// original null/default through the mock's backup storage
				// encoding; not the resource behaviour under test here.
				ImportStateVerifyIgnore: []string{"remote_type", "remote_path", "remote_retention", "remote_config"},
			},
		},
	})
}
