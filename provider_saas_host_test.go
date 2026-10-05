package main

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestIsSaaSHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"axonops.cloud", true},
		{"myorg.axonops.cloud", true},
		{"dash.axonops.cloud/myorg", true},
		{"MyOrg.AxonOps.Cloud", true},
		{"myorg.axonops.cloud.", true},
		{"myorg.axonops.cloud:443", true},
		{"axonopsdev.com", true},
		{"myorg.axonopsdev.com", true},
		{"myorg.axonopsdev.com:8443/dashboard", true},
		{"axonops.example.com", false},
		{"sergio.95.216.211.189.nip.io:8080", false},
		{"localhost:8080", false},
		{"127.0.0.1", false},
		{"notaxonops.cloud", false},
		{"axonops.cloud.example.com", false},
		{"evilaxonopsdev.com", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			if got := isSaaSHost(tc.host); got != tc.want {
				t.Errorf("isSaaSHost(%q) = %v, want %v", tc.host, got, tc.want)
			}
		})
	}
}

// TestAccProvider_selfHostedIgnoresDashboardProbe checks that a self-hosted
// server answering /dashboard/ with JSON, as it does with authentication
// enabled, is still reached at /api/v1 rather than /dashboard/api/v1.
func TestAccProvider_selfHostedIgnoresDashboardProbe(t *testing.T) {
	srv := newAccTestServer(t)
	srv.dashboardProbeJSON = true

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
resource "axonops_cassandra_agent_disconnect_tolerance" "t" {
  cluster_name = "ccluster"
  warn_timeout = "45s"
}
`,
				Check: resource.TestCheckResourceAttr("axonops_cassandra_agent_disconnect_tolerance.t", "warn_timeout", "45s"),
			},
		},
	})
}
