package main

import (
	"regexp"
	"testing"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// seedTestClusters registers clusters in the provider's org ("testorg") and
// one in another org, which must never be returned.
func seedTestClusters(srv *mockAxonOpsServer) {
	srv.seedCluster("testorg", "cassandra", "prod-cass", 0)
	srv.seedCluster("testorg", "cassandra", "dev-cass", 1)
	srv.seedCluster("testorg", "kafka", "prod-kafka", 2)
	srv.seedCluster("otherorg", "kafka", "foreign", 0)
}

func TestAccDataSourceClusters_list(t *testing.T) {
	srv := newAccTestServer(t)
	seedTestClusters(srv)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_clusters" "all" {}

data "axonops_clusters" "kafka" {
  type = "kafka"
}

data "axonops_clusters" "dse" {
  type = "dse"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.#", "3"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "names.#", "3"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.0.id", "cassandra/dev-cass"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.0.name", "dev-cass"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.0.type", "cassandra"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.0.status", "amber"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.1.name", "prod-cass"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.1.status", "green"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.2.id", "kafka/prod-kafka"),
					resource.TestCheckResourceAttr("data.axonops_clusters.all", "clusters.2.status", "red"),
					resource.TestCheckResourceAttr("data.axonops_clusters.kafka", "names.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_clusters.kafka", "names.0", "prod-kafka"),
					resource.TestCheckResourceAttr("data.axonops_clusters.dse", "clusters.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceClusters_invalidType(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_clusters" "bad" {
  type = "postgres"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

func TestAccDataSourceCluster_byName(t *testing.T) {
	srv := newAccTestServer(t)
	seedTestClusters(srv)
	srv.seedNodes("cassandra", "prod-cass",
		axonopsClient.ClusterNodeInfo{HostID: "h1", DC: "dc1", Active: true, Details: map[string]string{"comp_releaseVersion": "5.0.2"}},
		axonopsClient.ClusterNodeInfo{HostID: "h2", DC: "dc2", Active: true, Details: map[string]string{"comp_releaseVersion": "5.0.2"}},
		axonopsClient.ClusterNodeInfo{HostID: "h3", DC: "dc1", Active: false, Details: map[string]string{"comp_releaseVersion": "4.1.6"}},
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cluster" "c" {
  name = "prod-cass"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "id", "cassandra/prod-cass"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "type", "cassandra"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "status", "green"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "node_count", "3"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "active_node_count", "2"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "datacenters.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "datacenters.0", "dc1"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "versions.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "versions.0", "4.1.6"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "versions.1", "5.0.2"),
				),
			},
		},
	})
}

func TestAccDataSourceCluster_notFound(t *testing.T) {
	srv := newAccTestServer(t)
	seedTestClusters(srv)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Clusters of other orgs are not visible.
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cluster" "c" {
  name = "foreign"
}
`,
				ExpectError: regexp.MustCompile(`Cluster "foreign" not found`),
			},
			{
				// Name exists, but with a different type.
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cluster" "c" {
  name = "prod-kafka"
  type = "cassandra"
}
`,
				ExpectError: regexp.MustCompile(`cassandra cluster "prod-kafka" not found`),
			},
		},
	})
}

func TestAccDataSourceCluster_ambiguousNameRequiresType(t *testing.T) {
	srv := newAccTestServer(t)
	srv.seedCluster("testorg", "cassandra", "shared", 0)
	srv.seedCluster("testorg", "kafka", "shared", 1)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cluster" "c" {
  name = "shared"
}
`,
				ExpectError: regexp.MustCompile(`More than one cluster is named "shared"`),
			},
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cluster" "c" {
  name = "shared"
  type = "kafka"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "id", "kafka/shared"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "status", "amber"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "node_count", "0"),
					resource.TestCheckResourceAttr("data.axonops_cluster.c", "versions.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceClusterNodes_listAndFilter(t *testing.T) {
	srv := newAccTestServer(t)
	srv.seedNodes("cassandra", "prod-cass",
		axonopsClient.ClusterNodeInfo{HostID: "h3", HostIP: "10.0.1.3", DC: "dc2", Active: true,
			Details: map[string]string{"rack": "r1", "human_readable_identifier": "cass-3", "comp_releaseVersion": "5.0.2", "agent_version": "2.0.10"}},
		axonopsClient.ClusterNodeInfo{HostID: "h1", HostIP: "10.0.0.1", DC: "dc1", Active: true,
			Details: map[string]string{"rack": "r1", "human_readable_identifier": "cass-1", "comp_releaseVersion": "5.0.2", "agent_version": "2.0.10"}},
		axonopsClient.ClusterNodeInfo{HostID: "h2", HostIP: "10.0.0.2", DC: "dc1", Active: false,
			Details: map[string]string{"rack": "r2", "human_readable_identifier": "cass-2"}},
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cluster_nodes" "all" {
  cluster_name = "prod-cass"
  cluster_type = "cassandra"
}

data "axonops_cluster_nodes" "down" {
  cluster_name = "prod-cass"
  cluster_type = "cassandra"
  status       = "down"
}

data "axonops_cluster_nodes" "dc1_up" {
  cluster_name = "prod-cass"
  cluster_type = "cassandra"
  datacenter   = "dc1"
  status       = "up"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.#", "3"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "host_ids.#", "3"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "host_ids.0", "h1"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "host_ids.1", "h2"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "host_ids.2", "h3"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.0.host_ip", "10.0.0.1"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.0.hostname", "cass-1"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.0.datacenter", "dc1"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.0.rack", "r1"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.0.status", "up"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.0.version", "5.0.2"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.0.agent_version", "2.0.10"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.1.status", "down"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.all", "nodes.1.version", ""),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.down", "host_ids.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.down", "host_ids.0", "h2"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.dc1_up", "host_ids.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_cluster_nodes.dc1_up", "host_ids.0", "h1"),
				),
			},
		},
	})
}

func TestAccDataSourceClusterNodes_invalidStatus(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cluster_nodes" "bad" {
  cluster_name = "prod-cass"
  cluster_type = "cassandra"
  status       = "joining"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

func TestAccDataSourceCassandraKeyspaces_list(t *testing.T) {
	srv := newAccTestServer(t)
	srv.seedKeyspaces("cassandra", "prod-cass",
		axonopsClient.CassandraKeyspace{Name: "system_auth", System: true, ReplicationStrategy: "SimpleStrategy", ReplicationFactor: 1},
		axonopsClient.CassandraKeyspace{
			Name: "orders", ReplicationStrategy: "NetworkTopologyStrategy", ReplicationFactor: 3,
			ReplicationParams: `{"dc1":"3"}`,
			Tables:            []axonopsClient.CassandraTable{{Name: "by_id"}, {Name: "by_customer"}},
		},
		axonopsClient.CassandraKeyspace{Name: "audit", ReplicationStrategy: "NetworkTopologyStrategy", ReplicationFactor: 2},
	)
	srv.seedKeyspaces("dse", "dse-cluster", axonopsClient.CassandraKeyspace{Name: "search"})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cassandra_keyspaces" "user" {
  cluster_name = "prod-cass"
}

data "axonops_cassandra_keyspaces" "all" {
  cluster_name   = "prod-cass"
  include_system = true
}

data "axonops_cassandra_keyspaces" "dse" {
  cluster_name = "dse-cluster"
  cluster_type = "dse"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "names.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "names.0", "audit"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "names.1", "orders"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "keyspaces.1.replication_strategy", "NetworkTopologyStrategy"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "keyspaces.1.replication_factor", "3"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "keyspaces.1.replication_params", `{"dc1":"3"}`),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "keyspaces.1.system", "false"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "keyspaces.1.tables.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "keyspaces.1.tables.0", "by_customer"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.user", "keyspaces.0.tables.#", "0"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.all", "names.#", "3"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.all", "names.2", "system_auth"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.all", "keyspaces.2.system", "true"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.dse", "names.#", "1"),
					resource.TestCheckResourceAttr("data.axonops_cassandra_keyspaces.dse", "names.0", "search"),
				),
			},
		},
	})
}

func TestAccDataSourceCassandraKeyspaces_rejectsKafkaType(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_cassandra_keyspaces" "bad" {
  cluster_name = "prod-kafka"
  cluster_type = "kafka"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}

func TestAccDataSourceSchemas_listAndFilter(t *testing.T) {
	srv := newAccTestServer(t)

	resources := testAccProviderConfig(srv.URL()) + `
resource "axonops_schema" "orders" {
  cluster_name = "kcluster"
  subject      = "orders-value"
  schema       = jsonencode({ type = "record", name = "Order", fields = [] })
  schema_type  = "AVRO"
}

resource "axonops_schema" "payments" {
  cluster_name = "kcluster"
  subject      = "payments-value"
  schema       = jsonencode({ type = "record", name = "Payment", fields = [] })
  schema_type  = "AVRO"
}

resource "axonops_schema" "orders_key" {
  cluster_name = "kcluster"
  subject      = "orders-key"
  schema       = jsonencode({ type = "string" })
  schema_type  = "AVRO"
}
`
	lists := `
data "axonops_schemas" "all" {
  cluster_name = "kcluster"
  depends_on   = [axonops_schema.orders, axonops_schema.payments, axonops_schema.orders_key]
}

data "axonops_schemas" "values" {
  cluster_name  = "kcluster"
  subject_regex = "-value$"
  depends_on    = [axonops_schema.orders, axonops_schema.payments, axonops_schema.orders_key]
}

data "axonops_schemas" "with_deleted" {
  cluster_name    = "kcluster"
  include_deleted = true
  depends_on      = [axonops_schema.orders, axonops_schema.payments, axonops_schema.orders_key]
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: resources + lists,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_schemas.all", "subjects.#", "3"),
					resource.TestCheckResourceAttr("data.axonops_schemas.all", "subjects.0", "orders-key"),
					resource.TestCheckResourceAttr("data.axonops_schemas.all", "subjects.1", "orders-value"),
					resource.TestCheckResourceAttr("data.axonops_schemas.all", "subjects.2", "payments-value"),
					resource.TestCheckResourceAttr("data.axonops_schemas.values", "subjects.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_schemas.values", "subjects.0", "orders-value"),
				),
			},
			{
				PreConfig: func() { srv.softDeleteSubjectOutOfBand("kcluster", "orders-key") },
				Config:    resources + lists,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.axonops_schemas.all", "subjects.#", "2"),
					resource.TestCheckResourceAttr("data.axonops_schemas.with_deleted", "subjects.#", "3"),
				),
			},
		},
	})
}

func TestAccDataSourceSchemas_invalidRegex(t *testing.T) {
	srv := newAccTestServer(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(srv.URL()) + `
data "axonops_schemas" "bad" {
  cluster_name  = "kcluster"
  subject_regex = "orders-("
}
`,
				ExpectError: regexp.MustCompile(`Invalid Regular Expression`),
			},
		},
	})
}
