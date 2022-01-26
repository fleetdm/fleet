package redis_policy_set

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/fleet/policytest"
	"github.com/fleetdm/fleet/v4/server/test"
)

func TestRedisFailingPolicySet(t *testing.T) {
	for _, f := range []func(*testing.T, fleet.FailingPolicySet){
		policytest.RunFailingBasic,
		policytest.RunFailing1000hosts,
	} {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				store := setupRedis(t, false, false)
				f(t, store)
			})

			t.Run("cluster", func(t *testing.T) {
				store := setupRedis(t, true, true)
				f(t, store)
			})

			t.Run("cluster-no-redir", func(t *testing.T) {
				store := setupRedis(t, true, false)
				f(t, store)
			})
		})
	}
}

func setupRedis(t testing.TB, cluster, redir bool) *redisFailingPolicySet {
	pool := redistest.SetupRedis(t, t.Name(), cluster, redir, true)
	return NewFailingTest(t, pool)
}

func BenchmarkFailingPolicySetStandaloneP10H10(b *testing.B) {
	benchmarkFailingPolicySet(b, 10, 10, false)
}

func BenchmarkFailingPolicySetClusterP10H10(b *testing.B) {
	benchmarkFailingPolicySet(b, 10, 10, true)
}

func benchmarkFailingPolicySet(b *testing.B, policyCount, hostCount int, cluster bool) {
	s := setupRedis(b, cluster, false)
	for i := 0; i < b.N; i++ {
		runBenchmark(b, policyCount, hostCount, s)
	}
}

func runBenchmark(b *testing.B, policyCount, hostCount int, s *redisFailingPolicySet) {
	var wg sync.WaitGroup

	const checkInCount = 5

	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)

		for {
			select {
			case <-done:
				return
			default:
				sets, err := s.ListSets()
				if err != nil {
					b.Error(err)
				}
				for _, set := range sets {
					hosts, err := s.ListHosts(set)
					if err != nil {
						b.Error(err)
					}

					// simulate consumption of hosts
					time.Sleep(100 * time.Millisecond)

					err = s.RemoveHosts(set, hosts)
					if err != nil {
						b.Error(err)
					}
				}
			}
		}
	}()

	for hostID := 1; hostID < hostCount+1; hostID++ {
		hostID := uint(hostID)
		wg.Add(+1)
		go func() {
			defer wg.Done()

			for i := 0; i < checkInCount; i++ {
				for policyID := 1; policyID < policyCount+1; policyID++ {
					host := fleet.PolicySetHost{
						ID:       hostID,
						Hostname: fmt.Sprintf("test.hostname.%d", hostID),
					}
					var err error
					if rand.Float64() < 0.5 {
						err = s.AddHost(uint(policyID), host)
					} else {
						err = s.RemoveHosts(uint(policyID), []fleet.PolicySetHost{host})
					}
					if err != nil {
						b.Error(err)
						return
					}
				}
			}
		}()
	}

	wg.Wait()
	close(done)
	<-finished
}
