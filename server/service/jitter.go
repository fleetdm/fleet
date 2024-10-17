package service

import (
	"sync"
	"time"
)

// jitterHashTable implements a data structure that allows a fleet to generate a static jitter value
// that is properly balanced. Balance in this context means that hosts would be distributed uniformly
// across the total jitter time so there are no spikes.
// The way this structure works is as follows:
// Given an amount of buckets, we want to place hosts in buckets evenly. So we don't want bucket 0 to
// have 1000 hosts, and all the other buckets 0. If there were 1000 buckets, and 1000 hosts, we should
// end up with 1 per bucket.
// The total amount of online hosts is unknown, so first it assumes that amount of buckets >= amount
// of total hosts (maxCapacity of 1 per bucket). Once we have more hosts than buckets, then we
// increase the maxCapacity by 1 for all buckets, and start placing hosts.
// Hosts that have been placed in a bucket remain in that bucket for as long as the fleet instance is
// running.
// The preferred bucket for a host is the one at (host id % bucketCount). If that bucket is full, the
// next one will be tried. If all buckets are full, then capacity gets increased and the bucket
// selection process restarts.
// Once a bucket is found, the index for the bucket (going from 0 to bucketCount) will be the amount of
// minutes added to the host check in time.
// For example: at a 1hr interval, and the default 10% max jitter percent. That allows hosts to
// distribute within 6 minutes around the hour mark. We would have 6 buckets in that case.
// In the worst possible case that all hosts start at the same time, max jitter percent can be set to
// 100, and this method will distribute hosts evenly.
// The main caveat of this approach is that it works at the fleet instance. So depending on what
// instance gets chosen by the load balancer, the jitter might be different. However, load tests have
// shown that the distribution in practice is pretty balance even when all hosts try to check in at
// the same time.
type jitterHashTable struct {
	mu          sync.Mutex
	maxCapacity int
	bucketCount int
	buckets     map[int]int
	cache       map[uint]time.Duration
}

func newJitterHashTable(bucketCount int) *jitterHashTable {
	if bucketCount == 0 {
		bucketCount = 1
	}
	return &jitterHashTable{
		maxCapacity: 1,
		bucketCount: bucketCount,
		buckets:     make(map[int]int),
		cache:       make(map[uint]time.Duration),
	}
}

func (jh *jitterHashTable) jitterForHost(hostID uint) time.Duration {
	// if no jitter is configured just return 0
	if jh.bucketCount <= 1 {
		return 0
	}

	jh.mu.Lock()
	if jitter, ok := jh.cache[hostID]; ok {
		jh.mu.Unlock()
		return jitter
	}

	for i := 0; i < jh.bucketCount; i++ {
		possibleBucket := (int(hostID) + i) % jh.bucketCount //nolint:gosec // dismiss G115

		// if the next bucket has capacity, great!
		if jh.buckets[possibleBucket] < jh.maxCapacity {
			jh.buckets[possibleBucket]++
			jitter := time.Duration(possibleBucket) * time.Minute
			jh.cache[hostID] = jitter

			jh.mu.Unlock()
			return jitter
		}
	}

	// otherwise, bump the capacity and restart the process
	jh.maxCapacity++

	jh.mu.Unlock()
	return jh.jitterForHost(hostID)
}
