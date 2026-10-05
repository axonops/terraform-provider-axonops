# List every node in a cluster
data "axonops_cluster_nodes" "all" {
  cluster_name = "prod-cassandra"
  cluster_type = "cassandra"
}

output "node_ips" {
  value = [for n in data.axonops_cluster_nodes.all.nodes : n.host_ip]
}

# Only nodes whose agent is disconnected
data "axonops_cluster_nodes" "down" {
  cluster_name = "prod-cassandra"
  cluster_type = "cassandra"
  status       = "down"
}

output "down_nodes" {
  value = [for n in data.axonops_cluster_nodes.down.nodes : "${n.hostname} (${n.datacenter}/${n.rack})"]
}

# Only nodes in one data centre
data "axonops_cluster_nodes" "eu" {
  cluster_name = "prod-cassandra"
  cluster_type = "cassandra"
  datacenter   = "eu-west-1"
}

output "eu_host_ids" {
  value = data.axonops_cluster_nodes.eu.host_ids
}
