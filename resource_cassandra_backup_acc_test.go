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

// TestAccCassandraBackup_remoteConfigNoDrift guards against remote_config
// being refreshed from the API, which does not return it as sent: the
// post-apply plan must be empty, otherwise every apply deletes and recreates
// the backup.
func TestAccCassandraBackup_remoteConfigNoDrift(t *testing.T) {
	srv := newAccTestServer(t)
	config := testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_backup" "b" {
  cluster_name     = "ccluster"
  cluster_type     = "cassandra"
  tag              = "s3"
  local_retention  = "3d"
  datacenters      = ["dc1"]
  remote           = true
  remote_type      = "s3"
  remote_path      = "bucket/path"
  remote_retention = "90d"
  remote_config    = "access_key_id=AKIAEXAMPLE\nsecret_access_key=SECRET\nregion=us-east-1"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, PlanOnly: true},
		},
	})
}
