package main

import (
	"fmt"
	"regexp"
	"testing"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccAgentToleranceIs(srv *mockAxonOpsServer, warn, errTimeout string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		got := srv.agentTolerance[clusterKey("cassandra", "ccluster")]
		want := axonopsClient.AgentDisconnectionTolerance{WarnTimeout: warn, ErrorTimeout: errTimeout}
		if got == nil || *got != want {
			return fmt.Errorf("expected server tolerance %+v, got %+v", want, got)
		}
		return nil
	}
}

func TestAccCassandraAgentDisconnectTolerance_crud(t *testing.T) {
	srv := newAccTestServer(t)
	name := "axonops_cassandra_agent_disconnect_tolerance.t"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// Delete restores the AxonOps defaults.
		CheckDestroy: testAccAgentToleranceIs(srv, "30s", "1m"),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_agent_disconnect_tolerance" "t" {
  cluster_name = "ccluster"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "cluster_type", "cassandra"),
					resource.TestCheckResourceAttr(name, "warn_timeout", "30s"),
					resource.TestCheckResourceAttr(name, "error_timeout", "1m"),
					testAccAgentToleranceIs(srv, "30s", "1m"),
				),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_agent_disconnect_tolerance" "t" {
  cluster_name  = "ccluster"
  warn_timeout  = "2m"
  error_timeout = "10m"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "warn_timeout", "2m"),
					resource.TestCheckResourceAttr(name, "error_timeout", "10m"),
					testAccAgentToleranceIs(srv, "2m", "10m"),
				),
			},
			{
				ResourceName:                         name,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "cluster_name",
				ImportStateId:                        "cassandra/ccluster",
			},
		},
	})
}

// TestAccCassandraAgentDisconnectTolerance_drift checks that a change made
// outside Terraform is detected and reverted.
func TestAccCassandraAgentDisconnectTolerance_drift(t *testing.T) {
	srv := newAccTestServer(t)
	config := testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_agent_disconnect_tolerance" "t" {
  cluster_name  = "ccluster"
  warn_timeout  = "45s"
  error_timeout = "3m"
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
					srv.agentTolerance[clusterKey("cassandra", "ccluster")] = &axonopsClient.AgentDisconnectionTolerance{WarnTimeout: "5m", ErrorTimeout: "1h"}
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check:  testAccAgentToleranceIs(srv, "45s", "3m"),
			},
		},
	})
}

func TestAccCassandraAgentDisconnectTolerance_invalidInputs(t *testing.T) {
	srv := newAccTestServer(t)

	cases := []struct {
		name  string
		attrs string
		err   string
	}{
		{"warn not a duration", `warn_timeout = "soon"`, `must be a duration`},
		{"error without unit", `error_timeout = "60"`, `must be a duration`},
		{"kafka cluster_type", `cluster_type = "kafka"`, `value must be one of`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_agent_disconnect_tolerance" "t" {
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

func TestAccCassandraAgentDisconnectTolerance_invalidImportID(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_agent_disconnect_tolerance" "t" {
  cluster_name = "ccluster"
}
`,
				ResourceName:  "axonops_cassandra_agent_disconnect_tolerance.t",
				ImportState:   true,
				ImportStateId: "ccluster",
				ExpectError:   regexp.MustCompile(`cluster_type/cluster_name`),
			},
		},
	})
}
