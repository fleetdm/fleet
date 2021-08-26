package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out. http://stackoverflow.com/a/32843750/491710
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func TestQueryResultsStoreErrors(t *testing.T) {
	store, teardown := SetupRedisForTest(t)
	defer teardown()

	// Write with no subscriber
	err := store.WriteResult(
		fleet.DistributedQueryResult{
			DistributedQueryCampaignID: 9999,
			Rows:                       []map[string]string{{"bing": "fds"}},
			Host: fleet.Host{
				ID: 4,
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					UpdateTimestamp: fleet.UpdateTimestamp{
						UpdatedAt: time.Now().UTC(),
					},
				},
				DetailUpdatedAt: time.Now().UTC(),
			},
		},
	)
	assert.NotNil(t, err)
	castErr, ok := err.(Error)
	if assert.True(t, ok, "err should be pubsub.Error") {
		assert.True(t, castErr.NoSubscriber(), "NoSubscriber() should be true")
	}
}

func TestQueryResultsStore(t *testing.T) {
	store, teardown := SetupRedisForTest(t)
	defer teardown()

	// Test handling results for two campaigns in parallel
	campaign1 := fleet.DistributedQueryCampaign{ID: 1}

	ctx1, cancel1 := context.WithCancel(context.Background())
	channel1, err := store.ReadChannel(ctx1, campaign1)
	assert.Nil(t, err)

	expected1 := []fleet.DistributedQueryResult{
		{
			DistributedQueryCampaignID: 1,
			Rows:                       []map[string]string{{"foo": "bar"}},
			Host: fleet.Host{
				ID: 1,
				// Note these times need to be set to avoid
				// issues with roundtrip serializing the zero
				// time value. See https://goo.gl/CCEs8x
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					UpdateTimestamp: fleet.UpdateTimestamp{
						UpdatedAt: time.Now().UTC(),
					},
					CreateTimestamp: fleet.CreateTimestamp{
						CreatedAt: time.Now().UTC(),
					},
				},

				DetailUpdatedAt: time.Now().UTC(),
				SeenTime:        time.Now().UTC(),
			},
		},
		{
			DistributedQueryCampaignID: 1,
			Rows:                       []map[string]string{{"whoo": "wahh"}},
			Host: fleet.Host{
				ID: 3,
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					UpdateTimestamp: fleet.UpdateTimestamp{
						UpdatedAt: time.Now().UTC(),
					},
					CreateTimestamp: fleet.CreateTimestamp{
						CreatedAt: time.Now().UTC(),
					},
				},

				DetailUpdatedAt: time.Now().UTC(),
				SeenTime:        time.Now().UTC(),
			},
		},
		{
			DistributedQueryCampaignID: 1,
			Rows:                       []map[string]string{{"bing": "fds"}},
			Host: fleet.Host{
				ID: 4,
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					UpdateTimestamp: fleet.UpdateTimestamp{
						UpdatedAt: time.Now().UTC(),
					},
					CreateTimestamp: fleet.CreateTimestamp{
						CreatedAt: time.Now().UTC(),
					},
				},

				DetailUpdatedAt: time.Now().UTC(),
				SeenTime:        time.Now().UTC(),
			},
		},
	}

	campaign2 := fleet.DistributedQueryCampaign{ID: 2}

	ctx2, cancel2 := context.WithCancel(context.Background())
	channel2, err := store.ReadChannel(ctx2, campaign2)
	assert.Nil(t, err)

	expected2 := []fleet.DistributedQueryResult{
		{
			DistributedQueryCampaignID: 2,
			Rows:                       []map[string]string{{"tim": "tom"}},
			Host: fleet.Host{
				ID: 1,
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					UpdateTimestamp: fleet.UpdateTimestamp{
						UpdatedAt: time.Now().UTC(),
					},
					CreateTimestamp: fleet.CreateTimestamp{
						CreatedAt: time.Now().UTC(),
					},
				},

				DetailUpdatedAt: time.Now().UTC(),
				SeenTime:        time.Now().UTC(),
			},
		},
		{
			DistributedQueryCampaignID: 2,
			Rows:                       []map[string]string{{"slim": "slam"}},
			Host: fleet.Host{
				ID: 3,
				UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
					UpdateTimestamp: fleet.UpdateTimestamp{
						UpdatedAt: time.Now().UTC(),
					},
					CreateTimestamp: fleet.CreateTimestamp{
						CreatedAt: time.Now().UTC(),
					},
				},

				DetailUpdatedAt: time.Now().UTC(),
				SeenTime:        time.Now().UTC(),
			},
		},
	}

	var results1, results2 []fleet.DistributedQueryResult

	var readerWg, writerWg sync.WaitGroup

	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for res := range channel1 {
			switch res := res.(type) {
			case fleet.DistributedQueryResult:
				results1 = append(results1, res)
			}
		}

	}()
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for res := range channel2 {
			switch res := res.(type) {
			case fleet.DistributedQueryResult:
				results2 = append(results2, res)
			}
		}

	}()

	// Wait to ensure subscriptions are activated before writing
	time.Sleep(100 * time.Millisecond)

	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for _, res := range expected1 {
			assert.Nil(t, store.WriteResult(res))
		}
		time.Sleep(300 * time.Millisecond)
		cancel1()
	}()
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for _, res := range expected2 {
			assert.Nil(t, store.WriteResult(res))
		}
		time.Sleep(300 * time.Millisecond)
		cancel2()
	}()

	// wait with a timeout to ensure that the test can't hang
	if waitTimeout(&writerWg, 5*time.Second) {
		t.Error("Timed out waiting for writers to join")
	}
	if waitTimeout(&readerWg, 5*time.Second) {
		t.Error("Timed out waiting for readers to join")
	}

	assert.EqualValues(t, expected1, results1)
	assert.EqualValues(t, expected2, results2)
}
