package pubsub

import (
	"encoding/json"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
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

func functionName(f func(*testing.T, kolide.QueryResultStore)) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	elements := strings.Split(fullName, ".")
	return elements[len(elements)-1]
}

var testFunctions = [...]func(*testing.T, kolide.QueryResultStore){
	testQueryResultsStore,
	testQueryResultsStoreErrors,
}

func TestRedis(t *testing.T) {
	if _, ok := os.LookupEnv("REDIS_TEST"); !ok {
		t.SkipNow()
	}

	store, teardown := setupRedis(t)
	defer teardown()

	for _, f := range testFunctions {
		t.Run(functionName(f), func(t *testing.T) {
			f(t, store)
		})
	}
}

func TestInmem(t *testing.T) {
	for _, f := range testFunctions {
		t.Run(functionName(f), func(t *testing.T) {
			t.Parallel()
			store := NewInmemQueryResults()
			f(t, store)
		})
	}
}

func setupRedis(t *testing.T) (store *redisQueryResults, teardown func()) {
	var (
		addr     = "127.0.0.1:6379"
		password = ""
	)

	if a, ok := os.LookupEnv("REDIS_PORT_6379_TCP_ADDR"); ok {
		addr = a
	}

	store = NewRedisQueryResults(NewRedisPool(addr, password))

	_, err := store.pool.Get().Do("PING")
	require.Nil(t, err)

	teardown = func() {
		store.pool.Close()
	}

	return store, teardown
}

func testQueryResultsStoreErrors(t *testing.T, store kolide.QueryResultStore) {
	// Test handling results for two campaigns in parallel
	err := store.WriteResult(
		kolide.DistributedQueryResult{
			DistributedQueryCampaignID: 1,
			ResultJSON:                 json.RawMessage(`{"bing":"fds"}`),
			Host: kolide.Host{
				ID:               4,
				UpdatedAt:        time.Now(),
				DetailUpdateTime: time.Now(),
			},
		},
	)
	assert.NotNil(t, err)
}

func testQueryResultsStore(t *testing.T, store kolide.QueryResultStore) {
	// Test handling results for two campaigns in parallel
	campaign1 := kolide.DistributedQueryCampaign{ID: 1}

	ctx1, cancel1 := context.WithCancel(context.Background())
	channel1, err := store.ReadChannel(ctx1, campaign1)
	assert.Nil(t, err)

	expected1 := []kolide.DistributedQueryResult{
		kolide.DistributedQueryResult{
			DistributedQueryCampaignID: 1,
			ResultJSON:                 json.RawMessage(`{"foo":"bar"}`),
			Host: kolide.Host{
				ID: 1,
				// Note these times need to be set to avoid
				// issues with roundtrip serializing the zero
				// time value. See https://goo.gl/CCEs8x
				UpdatedAt:        time.Now(),
				DetailUpdateTime: time.Now(),
			},
		},
		kolide.DistributedQueryResult{
			DistributedQueryCampaignID: 1,
			ResultJSON:                 json.RawMessage(`{"whoo":"wahh"}`),
			Host: kolide.Host{
				ID:               3,
				UpdatedAt:        time.Now(),
				DetailUpdateTime: time.Now(),
			},
		},
		kolide.DistributedQueryResult{
			DistributedQueryCampaignID: 1,
			ResultJSON:                 json.RawMessage(`{"bing":"fds"}`),
			Host: kolide.Host{
				ID:               4,
				UpdatedAt:        time.Now(),
				DetailUpdateTime: time.Now(),
			},
		},
	}

	campaign2 := kolide.DistributedQueryCampaign{ID: 2}

	ctx2, cancel2 := context.WithCancel(context.Background())
	channel2, err := store.ReadChannel(ctx2, campaign2)
	assert.Nil(t, err)

	expected2 := []kolide.DistributedQueryResult{
		kolide.DistributedQueryResult{
			DistributedQueryCampaignID: 2,
			ResultJSON:                 json.RawMessage(`{"tim":"tom"}`),
			Host: kolide.Host{
				ID:               1,
				UpdatedAt:        time.Now(),
				DetailUpdateTime: time.Now(),
			},
		},
		kolide.DistributedQueryResult{
			DistributedQueryCampaignID: 2,
			ResultJSON:                 json.RawMessage(`{"slim":"slam"}`),
			Host: kolide.Host{
				ID:               3,
				UpdatedAt:        time.Now(),
				DetailUpdateTime: time.Now(),
			},
		},
	}

	var results1, results2 []kolide.DistributedQueryResult

	var readerWg, writerWg sync.WaitGroup

	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for res := range channel1 {
			switch res := res.(type) {
			case kolide.DistributedQueryResult:
				results1 = append(results1, res)
			}
		}

	}()
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for res := range channel2 {
			switch res := res.(type) {
			case kolide.DistributedQueryResult:
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
