# Cassandra Service Checks (Healthchecks)
# Ported from the axonops-ansible-collection Cassandra service check examples.
# Shell checks reference external scripts — see the ansible collection for
# full inline script implementations.


# ── TCP Checks ────────────────────────────────────────────────────────

# Verify the CQL native transport port is accepting connections
resource "axonops_healthcheck_tcp" "cql_port" {
  cluster_name = var.cluster_name
  name         = "cql_client_port"
  tcp          = "{{.comp_listen_address}}:{{.comp_native_transport_port}}"
  interval     = "3m"
  timeout      = "1m"
}


# ── Shell Checks ──────────────────────────────────────────────────────

# Detect DOWN nodes via nodetool status
resource "axonops_healthcheck_shell" "node_down_check" {
  cluster_name = var.cluster_name
  name         = "Check for node DOWN"
  script       = "/usr/local/bin/cassandra-check-node-down.sh"
  shell        = "/bin/bash"
  interval     = "15m"
  timeout      = "2m"
}

# Detect reboot-required flag (Debian/Ubuntu)
resource "axonops_healthcheck_shell" "reboot_required" {
  cluster_name = var.cluster_name
  name         = "Debian / Ubuntu - Check host needs reboot"
  script       = "/usr/local/bin/check-reboot-required.sh"
  shell        = "/bin/bash"
  interval     = "12h"
  timeout      = "1m"
}

# Check for scheduled AWS maintenance events
resource "axonops_healthcheck_shell" "aws_maintenance_events" {
  cluster_name = var.cluster_name
  name         = "Check AWS events"
  script       = "/usr/local/bin/check-aws-events.py"
  shell        = "/usr/bin/python3"
  interval     = "12h"
  timeout      = "1m"
}

# Detect schema disagreements across the cluster
resource "axonops_healthcheck_shell" "schema_disagreements" {
  cluster_name = var.cluster_name
  name         = "Check for schema disagreements"
  script       = "/usr/local/bin/cassandra-check-schema.sh"
  shell        = "/bin/bash"
  interval     = "1d"
  timeout      = "1m"
}

# Verify Cassandra data directory ownership
resource "axonops_healthcheck_shell" "data_ownership" {
  cluster_name = var.cluster_name
  name         = "Checkout cassandra data ownership"
  script       = "/usr/local/bin/cassandra-check-data-ownership.py"
  shell        = "/usr/bin/python3"
  interval     = "1d"
  timeout      = "3m"
}

# Check for excessive commitlog archive backlog
resource "axonops_healthcheck_shell" "commitlog_archives" {
  cluster_name = var.cluster_name
  name         = "Check for commitlog archives"
  script       = "/usr/local/bin/cassandra-check-commitlog-archives.sh"
  shell        = "/bin/bash"
  interval     = "12h"
  timeout      = "1m"
}

# Verify TLS certificate expiry on the CQL port
resource "axonops_healthcheck_shell" "ssl_certificate" {
  cluster_name = var.cluster_name
  name         = "SSL certificate check"
  script       = "/usr/local/bin/cassandra-check-ssl-cert.sh"
  shell        = "/bin/bash"
  interval     = "12h"
  timeout      = "1m"
}

# Run CQL read queries at each consistency level to verify cluster health
resource "axonops_healthcheck_shell" "cql_consistency_test" {
  cluster_name = var.cluster_name
  name         = "Cassandra CQL Consistency Level Test Script"
  script       = "/usr/local/bin/cassandra-cql-consistency-test.sh"
  shell        = "/bin/bash"
  interval     = "12h"
  timeout      = "1m"
}
