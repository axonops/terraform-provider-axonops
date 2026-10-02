package main

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// validClusterTypes lists the cluster types supported by the AxonOps API.
var validClusterTypes = []string{"cassandra", "kafka", "dse"}

// clusterTypeValidator restricts cluster_type attributes to supported values.
func clusterTypeValidator() validator.String {
	return stringvalidator.OneOf(validClusterTypes...)
}

// cassandraOnlyClusterTypes lists the cluster types supported by
// Cassandra/DSE-specific features that do not apply to Kafka clusters.
var cassandraOnlyClusterTypes = []string{"cassandra", "dse"}

// cassandraOnlyClusterTypeValidator restricts cluster_type attributes to
// Cassandra/DSE, for resources that have no meaning on a Kafka cluster.
func cassandraOnlyClusterTypeValidator() validator.String {
	return stringvalidator.OneOf(cassandraOnlyClusterTypes...)
}

// durationRegex matches AxonOps duration strings such as "30s", "5m", "1h",
// "7d", "2w" or compound values like "1h30m".
var durationRegex = regexp.MustCompile(`^([0-9]+(ms|s|m|h|d|w|y))+$`)

// durationValidator validates AxonOps duration strings.
func durationValidator() validator.String {
	return stringvalidator.RegexMatches(durationRegex, `must be a duration such as "30s", "5m", "1h" or "7d"`)
}

// cronRegex matches a standard 5-field cron expression, optionally with a
// leading seconds field (6 fields), or an @-descriptor such as @daily.
var cronRegex = regexp.MustCompile(`^(@(annually|yearly|monthly|weekly|daily|midnight|hourly)|@every\s+\S+|(\S+\s+){4,5}\S+)$`)

// cronValidator validates cron expressions.
func cronValidator() validator.String {
	return stringvalidator.RegexMatches(cronRegex, "must be a 5- or 6-field cron expression or an @-descriptor such as @daily")
}
