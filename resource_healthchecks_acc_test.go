package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccHealthchecks_parallelCreatesAllPersist creates several healthchecks
// of mixed types (http, tcp, shell) on the same cluster within a single
// apply. Since all three types share one document server-side
// (GetHealthchecks/UpdateHealthchecks), concurrent Terraform creates race on
// a read-modify-write unless serialized (see cluster_lock.go). This verifies
// none of the three silently clobbers the others.
func TestAccHealthchecks_parallelCreatesAllPersist(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_healthcheck_http" "h" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "http-check"
  url          = "http://localhost:8080/health"
  # A non-empty headers map sidesteps an unrelated known issue where an
  # empty map round-trips through the "headers,omitempty" JSON tag as null
  # and produces a perpetual diff on refresh (see final test report).
  headers = {
    "X-Check" = "1"
  }
}

resource "axonops_healthcheck_tcp" "t" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "tcp-check"
  tcp          = "0.0.0.0:9042"
}

resource "axonops_healthcheck_shell" "s" {
  cluster_name = "ccluster"
  cluster_type = "cassandra"
  name         = "shell-check"
  script       = "/usr/bin/true"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("axonops_healthcheck_http.h", "id"),
					resource.TestCheckResourceAttrSet("axonops_healthcheck_tcp.t", "id"),
					resource.TestCheckResourceAttrSet("axonops_healthcheck_shell.s", "id"),
					testAccCheckHealthchecksDocumentHasAll(srv, "cassandra", "ccluster", "http-check", "tcp-check", "shell-check"),
				),
			},
		},
	})
}

// testAccCheckHealthchecksDocumentHasAll inspects the mock server's document
// directly (bypassing Terraform state) to confirm all three healthchecks
// really persisted server-side, catching a lost-update race even if
// Terraform's own state happens to look consistent.
func testAccCheckHealthchecksDocumentHasAll(srv *mockAxonOpsServer, clusterType, clusterName string, httpName, tcpName, shellName string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()

		doc := srv.healthchecks[clusterKey(clusterType, clusterName)]
		if doc == nil {
			return fmt.Errorf("no healthchecks document found for %s/%s", clusterType, clusterName)
		}

		hasHTTP := false
		for _, c := range doc.HTTPChecks {
			if c.Name == httpName {
				hasHTTP = true
			}
		}
		hasTCP := false
		for _, c := range doc.TCPChecks {
			if c.Name == tcpName {
				hasTCP = true
			}
		}
		hasShell := false
		for _, c := range doc.ShellChecks {
			if c.Name == shellName {
				hasShell = true
			}
		}

		if !hasHTTP || !hasTCP || !hasShell {
			return fmt.Errorf("healthchecks document missing entries: http=%v tcp=%v shell=%v (document: %+v)", hasHTTP, hasTCP, hasShell, doc)
		}
		return nil
	}
}
