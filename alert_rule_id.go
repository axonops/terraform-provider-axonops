package main

import (
	"strings"

	"github.com/google/uuid"
)

// deterministicAlertRuleID returns a UUID v5 derived deterministically from
// the combination of org, cluster type, cluster name, alert name, and resource
// kind ("log" or "metric"). The same inputs always produce the same UUID,
// making Create idempotent against state loss and against retries that follow
// an API call which succeeded server-side but failed client-side. Two rules
// configured with the same name on the same cluster of the same type collide
// — that is intentional, because such a configuration would otherwise produce
// two indistinguishable rules in AxonOps.
func deterministicAlertRuleID(orgID, clusterType, clusterName, alertName, kind string) string {
	key := strings.Join([]string{orgID, clusterType, clusterName, alertName, kind}, "/")
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(key)).String()
}
