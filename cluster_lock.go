package main

import (
	"fmt"
	"sync"
)

// clusterLocks holds one *sync.Mutex per (kind, cluster_type, cluster_name)
// key. It serializes the read-modify-write (GET whole doc / mutate / PUT
// whole doc) sequences used by the healthcheck and log collector resources,
// which all share a single document per cluster. Without this, concurrent
// Terraform operations against sibling resources of the same document (e.g.
// two httpHealthcheckResource instances, or an httpHealthcheckResource and a
// tcpHealthcheckResource on the same cluster) can race: both read the same
// starting document, each applies its own mutation, and whichever PUT lands
// last silently discards the other's change (a classic lost update).
//
// NOTE: this only protects against concurrent operations within a single
// provider process (e.g. Terraform's default parallelism within one `apply`
// run). It does NOT protect against concurrent writers across separate
// processes/machines (e.g. two CI pipelines applying at once, or a human
// editing via the AxonOps UI at the same time); that requires optimistic
// concurrency control (ETags/versioning) on the server side, which the
// AxonOps API does not currently expose.
var clusterLocks sync.Map // map[string]*sync.Mutex

// lockCluster acquires the mutex for the given (kind, clusterType,
// clusterName) key and returns a function that releases it. kind should be
// a short constant identifying the shared document, e.g. "healthchecks" or
// "logcollectors". Callers must defer the returned unlock function
// immediately:
//
//	unlock := lockCluster("healthchecks", clusterType, clusterName)
//	defer unlock()
func lockCluster(kind, clusterType, clusterName string) func() {
	key := fmt.Sprintf("%s/%s/%s", kind, clusterType, clusterName)

	value, _ := clusterLocks.LoadOrStore(key, &sync.Mutex{})
	mu := value.(*sync.Mutex)

	mu.Lock()
	return mu.Unlock
}
