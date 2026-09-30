package main

// provider_test.go wires the provider under test to terraform-plugin-testing
// for acceptance tests (TF_ACC=1 go test ./...). It configures the provider
// to talk to an in-process mockAxonOpsServer instead of the real AxonOps API,
// so CRUD can be exercised without credentials or network access.

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories is the map terraform-plugin-testing uses to
// instantiate the provider under test. main.go serves the provider over
// protocol v6 (providerserver.Serve without an explicit ProtocolVersion
// defaults to 6), so tests must use the matching NewProtocol6WithError.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"axonops": providerserver.NewProtocol6WithError(New()()),
}

// testAccProviderConfig returns a provider configuration block pointed at
// the given mock server host, using the non-SAML on-prem layout (the mock's
// /dashboard/ probe always answers 404).
func testAccProviderConfig(host string) string {
	return `
provider "axonops" {
  org_id           = "testorg"
  axonops_host     = "` + host + `"
  axonops_protocol = "http"
  api_key          = "test-api-key"
  token_type       = "Bearer"
}
`
}

// newAccTestServer starts a fresh mock AxonOps API server for a single test
// and returns it along with the provider configuration block that targets
// it. Each acceptance test gets its own isolated server instance.
func newAccTestServer(t *testing.T) *mockAxonOpsServer {
	t.Helper()
	return newMockAxonOpsServer(t)
}

// testAccAlertRuleImportStateID builds the cluster_type/cluster_name/id
// import ID for metric/log alert rule resources from their own prior-step
// state, since the alert rule ID is assigned server-side.
func testAccAlertRuleImportStateID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceName)
		}
		attrs := rs.Primary.Attributes
		return fmt.Sprintf("%s/%s/%s", attrs["cluster_type"], attrs["cluster_name"], attrs["id"]), nil
	}
}
