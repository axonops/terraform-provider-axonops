package main

import (
	"fmt"
	"regexp"
	"testing"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccCommitLogSettingsDestroyed(srv *mockAxonOpsServer) resource.TestCheckFunc {
	return func(*terraform.State) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		if n := len(srv.commitLogSettings[clusterKey("cassandra", "ccluster")]); n != 0 {
			return fmt.Errorf("expected no commitlog archive settings after destroy, got %d", n)
		}
		return nil
	}
}

func TestAccCassandraCommitlogSettings_crud(t *testing.T) {
	srv := newAccTestServer(t)
	name := "axonops_cassandra_commitlog_settings.c"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCommitLogSettingsDestroyed(srv),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name = "ccluster"
  datacenters  = ["dc1", "dc2"]
  remote_type  = "local"
  remote_path  = "/backups/commitlogs"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "cluster_type", "cassandra"),
					resource.TestCheckResourceAttr(name, "datacenters.#", "2"),
					resource.TestCheckResourceAttr(name, "remote_type", "local"),
					resource.TestCheckResourceAttr(name, "remote_retention", "60d"),
					resource.TestCheckResourceAttr(name, "timeout", "10h"),
					resource.TestCheckResourceAttr(name, "transfers", "0"),
					func(*terraform.State) error {
						srv.mu.Lock()
						defer srv.mu.Unlock()
						got := srv.commitLogSettings[clusterKey("cassandra", "ccluster")]
						if len(got) != 1 || got[0].RemoteConfig != "type = local" {
							return fmt.Errorf("expected one config with remoteConfig %q, got %+v", "type = local", got)
						}
						return nil
					},
				),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name     = "ccluster"
  datacenters      = ["dc1", "dc2"]
  remote_type      = "s3"
  remote_path      = "bucket/commitlogs"
  remote_retention = "30d"
  remote_config    = "type = s3\nprovider = AWS\nregion = eu-west-1\nenv_auth = true"
  timeout          = "2h"
  transfers        = 4
  bw_limit         = "10M"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "remote_type", "s3"),
					resource.TestCheckResourceAttr(name, "remote_path", "bucket/commitlogs"),
					resource.TestCheckResourceAttr(name, "remote_retention", "30d"),
					resource.TestCheckResourceAttr(name, "timeout", "2h"),
					resource.TestCheckResourceAttr(name, "transfers", "4"),
					resource.TestCheckResourceAttr(name, "bw_limit", "10M"),
				),
			},
			{
				// remote_config is masked by the API: refreshing must not
				// produce a diff.
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name     = "ccluster"
  datacenters      = ["dc1", "dc2"]
  remote_type      = "s3"
  remote_path      = "bucket/commitlogs"
  remote_retention = "30d"
  remote_config    = "type = s3\nprovider = AWS\nregion = eu-west-1\nenv_auth = true"
  timeout          = "2h"
  transfers        = 4
  bw_limit         = "10M"
}
`,
				PlanOnly: true,
			},
			{
				ResourceName:                         name,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "cluster_name",
				ImportStateId:                        "cassandra/ccluster/dc2",
				ImportStateVerifyIgnore:              []string{"remote_config"},
			},
		},
	})
}

// TestAccCassandraCommitlogSettings_removedOutOfBand checks that settings
// deleted outside Terraform are recreated on the next apply.
func TestAccCassandraCommitlogSettings_removedOutOfBand(t *testing.T) {
	srv := newAccTestServer(t)
	config := testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name = "ccluster"
  datacenters  = ["dc1"]
  remote_type  = "local"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					srv.mu.Lock()
					defer srv.mu.Unlock()
					delete(srv.commitLogSettings, clusterKey("cassandra", "ccluster"))
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{Config: config},
		},
	})
}

func TestAccCassandraCommitlogSettings_existingDatacenterRejected(t *testing.T) {
	srv := newAccTestServer(t)
	srv.commitLogSettings[clusterKey("cassandra", "ccluster")] = []axonopsClient.CommitLogArchiveSettings{
		{Datacenters: []string{"dc1"}, RemoteType: "local", RemoteRetentionDuration: "60d", Timeout: "10h"},
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name = "ccluster"
  datacenters  = ["dc2", "dc1"]
  remote_type  = "local"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Datacenter "dc1".*already has commitlog archive.*cassandra/ccluster/dc1`),
			},
		},
	})
}

func TestAccCassandraCommitlogSettings_invalidInputs(t *testing.T) {
	srv := newAccTestServer(t)

	cases := []struct {
		name  string
		attrs string
		err   string
	}{
		{"empty datacenters", `datacenters = []
  remote_type = "local"`, `at least 1`},
		{"duplicate datacenters", `datacenters = ["dc1", "dc1"]
  remote_type = "local"`, `duplicate values`},
		{"unknown remote_type", `datacenters = ["dc1"]
  remote_type = "ftp"`, `value must be one of`},
		{"bad retention", `datacenters = ["dc1"]
  remote_type = "local"
  remote_retention = "sixty days"`, `must be a duration`},
		{"negative transfers", `datacenters = ["dc1"]
  remote_type = "local"
  transfers = -1`, `at least 0`},
		{"kafka cluster_type", `datacenters = ["dc1"]
  remote_type  = "local"
  cluster_type = "kafka"`, `value must be one of`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name = "ccluster"
  ` + tc.attrs + `
}
`,
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(tc.err),
					},
				},
			})
		})
	}
}

func TestAccCassandraCommitlogSettings_invalidImportID(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name = "ccluster"
  datacenters  = ["dc1"]
  remote_type  = "local"
}
`,
				ResourceName:  "axonops_cassandra_commitlog_settings.c",
				ImportState:   true,
				ImportStateId: "cassandra/ccluster",
				ExpectError:   regexp.MustCompile(`cluster_type/cluster_name/datacenter`),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_commitlog_settings" "c" {
  cluster_name = "ccluster"
  datacenters  = ["dc1"]
  remote_type  = "local"
}
`,
				ResourceName:  "axonops_cassandra_commitlog_settings.c",
				ImportState:   true,
				ImportStateId: "cassandra/ccluster/dc9",
				ExpectError:   regexp.MustCompile(`No commitlog archive settings for datacenter dc9`),
			},
		},
	})
}

func TestCommitLogRemoteConfig(t *testing.T) {
	cases := []struct {
		name, remoteType, config, want string
	}{
		{"empty config gets type", "local", "", "type = local"},
		{"blank config gets type", "local", "  \n", "type = local"},
		{"type prepended", "s3", "region = us-east-1", "type = s3\nregion = us-east-1"},
		{"existing type kept", "s3", "region = us-east-1\ntype = s3", "region = us-east-1\ntype = s3"},
		{"type without spaces kept", "sftp", "type=sftp\nhost=h", "type=sftp\nhost=h"},
		{"type-like key is not type", "s3", "storage_type = x", "type = s3\nstorage_type = x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := commitLogRemoteConfig(tc.remoteType, tc.config); got != tc.want {
				t.Errorf("commitLogRemoteConfig(%q, %q) = %q, want %q", tc.remoteType, tc.config, got, tc.want)
			}
		})
	}
}
