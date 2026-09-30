package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestLockCluster_SameKeySerializes proves that concurrent lockCluster
// callers using the same key never run their critical sections at the same
// time.
func TestLockCluster_SameKeySerializes(t *testing.T) {
	const goroutines = 20

	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			unlock := lockCluster("healthchecks", "cassandra", "shared-cluster")
			defer unlock()

			n := atomic.AddInt32(&active, 1)
			for {
				cur := atomic.LoadInt32(&maxActive)
				if n <= cur {
					break
				}
				if atomic.CompareAndSwapInt32(&maxActive, cur, n) {
					break
				}
			}

			time.Sleep(2 * time.Millisecond)
			atomic.AddInt32(&active, -1)
		}()
	}

	wg.Wait()

	if maxActive != 1 {
		t.Fatalf("expected at most 1 concurrent holder of the same lock key, got %d", maxActive)
	}
}

// TestLockCluster_DifferentKeysDoNotBlock proves that lockCluster calls with
// different keys can proceed concurrently instead of serializing on a single
// global lock.
func TestLockCluster_DifferentKeysDoNotBlock(t *testing.T) {
	const goroutines = 10

	var wg sync.WaitGroup
	release := make(chan struct{})

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		clusterName := "cluster"
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			unlock := lockCluster("healthchecks", "cassandra", clusterName+string(rune('a'+idx)))
			defer unlock()

			<-release
		}(i)
	}

	// Give every goroutine a chance to acquire its (distinct) lock before we
	// release them all at once. If locks were serialized on a single global
	// mutex, only one goroutine could be holding its lock at a time and this
	// would not be able to complete quickly once release is closed.
	time.Sleep(20 * time.Millisecond)
	close(release)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for goroutines with different lock keys to complete")
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("goroutines with different keys took too long (%s), suggesting they serialized", elapsed)
	}
}
