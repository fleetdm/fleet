package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	runTest := func(t *testing.T, store *redisQueryResults) {
		result := fleet.DistributedQueryResult{
			DistributedQueryCampaignID: 9999,
			Rows:                       []map[string]string{{"bing": "fds"}},
			Host: fleet.ResultHostData{
				ID: 4,
			},
		}

		// Write with no subscriber
		err := store.WriteResult(result)
		require.Error(t, err)
		castErr, ok := err.(Error)
		if assert.True(t, ok, "err should be pubsub.Error") {
			assert.True(t, castErr.NoSubscriber(), "NoSubscriber() should be true")
		}

		// Write with one subscriber, force it to bind to a different node if
		// this is a cluster, so we don't rely on publishing/subscribing on the
		// same nodes.
		conn := redis.ReadOnlyConn(store.pool, store.pool.Get())
		defer conn.Close()
		err = redis.BindConn(store.pool, conn, "ZZZ")
		require.NoError(t, err)

		psc := &redigo.PubSubConn{Conn: conn}
		pubSubName := pubSubForID(9999)
		require.NoError(t, psc.Subscribe(pubSubName))

		// wait for subscribed confirmation
		start := time.Now()
		var loopOk bool
	loop:
		for time.Since(start) < 2*time.Second {
			msg := psc.Receive()
			switch msg := msg.(type) {
			case redigo.Subscription:
				require.Equal(t, msg.Count, 1)
				loopOk = true
				break loop
			}
		}
		require.True(t, loopOk, "timed out")

		err = store.WriteResult(result)
		require.NoError(t, err)
	}

	t.Run("standalone", func(t *testing.T) {
		store := SetupRedisForTest(t, false, false)
		runTest(t, store)
	})

	t.Run("cluster", func(t *testing.T) {
		store := SetupRedisForTest(t, true, true)
		runTest(t, store)
	})
}

func TestQueryResultsStore(t *testing.T) {
	runTest := func(t *testing.T, store *redisQueryResults) {
		// Test handling results for two campaigns in parallel
		campaign1 := fleet.DistributedQueryCampaign{ID: 1}

		ctx1, cancel1 := context.WithCancel(context.Background())
		channel1, err := store.ReadChannel(ctx1, campaign1)
		require.NoError(t, err)

		expected1 := []fleet.DistributedQueryResult{
			{
				DistributedQueryCampaignID: 1,
				Rows:                       []map[string]string{{"foo": "bar"}},
				Host: fleet.ResultHostData{
					ID: 1,
				},
			},
			{
				DistributedQueryCampaignID: 1,
				Rows:                       []map[string]string{{"whoo": "wahh"}},
				Host: fleet.ResultHostData{
					ID: 3,
				},
			},
			{
				DistributedQueryCampaignID: 1,
				Rows:                       []map[string]string{{"bing": "fds"}},
				Host: fleet.ResultHostData{
					ID: 4,
				},
			},
		}

		campaign2 := fleet.DistributedQueryCampaign{ID: 2}

		ctx2, cancel2 := context.WithCancel(context.Background())
		channel2, err := store.ReadChannel(ctx2, campaign2)
		require.NoError(t, err)

		expected2 := []fleet.DistributedQueryResult{
			{
				DistributedQueryCampaignID: 2,
				Rows:                       []map[string]string{{"tim": "tom"}},
				Host: fleet.ResultHostData{
					ID: 1,
				},
			},
			{
				DistributedQueryCampaignID: 2,
				Rows:                       []map[string]string{{"slim": "slam"}},
				Host: fleet.ResultHostData{
					ID: 3,
				},
			},
		}

		var results1, results2 []fleet.DistributedQueryResult

		var readerWg, writerWg sync.WaitGroup

		readerWg.Add(1)
		go func() {
			defer readerWg.Done()
			for res := range channel1 {
				switch res := res.(type) { //nolint:gocritic // ignore singleCaseSwitch
				case fleet.DistributedQueryResult:
					results1 = append(results1, res)
				}
			}
		}()
		readerWg.Add(1)
		go func() {
			defer readerWg.Done()
			for res := range channel2 {
				switch res := res.(type) { //nolint:gocritic // ignore singleCaseSwitch
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

	t.Run("standalone", func(t *testing.T) {
		store := SetupRedisForTest(t, false, false)
		runTest(t, store)
	})

	t.Run("cluster", func(t *testing.T) {
		store := SetupRedisForTest(t, true, true)
		runTest(t, store)
	})
}
