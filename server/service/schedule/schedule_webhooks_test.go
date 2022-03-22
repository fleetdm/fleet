package schedule

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

// TODO: fix races?
func TestCronWebhooks(t *testing.T) {
	ds := new(mock.Store)

	endpointCalled := int32(0)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&endpointCalled, 1)
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				HostStatusWebhook: fleet.HostStatusWebhookSettings{
					Enable:         true,
					DestinationURL: ts.URL,
					HostPercentage: 43,
					DaysCount:      2,
				},
				Interval: fleet.Duration{Duration: 2 * time.Second},
			},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	calledOnce := make(chan struct{})
	calledTwice := make(chan struct{})
	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, daysCount int) (int, int, error) {
		defer func() {
			select {
			case <-calledOnce:
				select {
				case <-calledTwice:
				default:
					close(calledTwice)
				}
			default:
				close(calledOnce)
			}
		}()
		return 10, 6, nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)

	webhooksLogger := log.NewNopLogger()
	webhooksInterval := appConfig.WebhookSettings.Interval.ValueOr(30 * time.Second)
	fmt.Println(webhooksInterval)
	webhooks, err := New(ctx, "webhooks", "test_instance", webhooksInterval, ds, webhooksLogger)
	require.NoError(t, err)

	webhooks.setConfigInterval(5 * time.Minute)
	webhooks.SetConfigCheck(SetWebhooksConfigCheck(ctx, ds, webhooksLogger))
	webhooks.AddJob("cron_webhooks", func(ctx context.Context) (interface{}, error) {
		return DoWebhooks(ctx, ds, webhooksLogger, service.NewMemFailingPolicySet())
	}, func(interface{}, error) {})

	<-calledOnce
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(1), atomic.LoadInt32(&endpointCalled))
	<-calledTwice
	time.Sleep(1 * time.Second)
	assert.GreaterOrEqual(t, int32(2), atomic.LoadInt32(&endpointCalled))
}

// TestCronWebhooksLockDuration tests that the Lock method is being called with a duration equal to the schedule interval
// TODO: should the lock duration be the schedule interval or always be set to one hour (see #3584)?
// TODO: fix races
func TestCronWebhooksLockDuration(t *testing.T) {
	ds := new(mock.Store)
	interval := 1 * time.Second

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				Interval: fleet.Duration{Duration: interval},
			},
		}, nil
	}
	hostStatus := make(chan struct{})
	hostStatusClosed := false
	failingPolicies := make(chan struct{})
	failingPoliciesClosed := false
	unknownName := false
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		if expiration != interval {
			return false, nil
		}
		switch name {
		case "webhooks":
			if !hostStatusClosed {
				close(hostStatus)
				hostStatusClosed = true
			}
			if !failingPoliciesClosed {
				close(failingPolicies)
				failingPoliciesClosed = true
			}

		default:
			unknownName = true
		}
		return true, nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)

	webhooksLogger := log.NewNopLogger()
	webhooksInterval := appConfig.WebhookSettings.Interval.ValueOr(30 * time.Second)
	fmt.Println(webhooksInterval)
	webhooks, err := New(ctx, "webhooks", "test_instance", webhooksInterval, ds, webhooksLogger)
	require.NoError(t, err)

	webhooks.SetConfigCheck(SetWebhooksConfigCheck(ctx, ds, webhooksLogger))
	webhooks.AddJob("cron_webhooks", func(ctx context.Context) (interface{}, error) {
		return DoWebhooks(ctx, ds, webhooksLogger, service.NewMemFailingPolicySet())
	}, func(interface{}, error) {})

	select {
	case <-failingPolicies:
	case <-time.After(5 * time.Second):
		t.Error("failing policies timeout")
	}
	select {
	case <-hostStatus:
	case <-time.After(5 * time.Second):
		t.Error("host status timeout")
	}
	require.False(t, unknownName)
}

// TODO: fix races
func TestCronWebhooksIntervalChange(t *testing.T) {
	ds := new(mock.Store)

	interval := struct {
		sync.Mutex
		value time.Duration
	}{
		value: 5 * time.Hour,
	}
	configLoaded := make(chan struct{}, 1)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		select {
		case configLoaded <- struct{}{}:
		default:
			// OK
		}

		interval.Lock()
		defer interval.Unlock()

		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				Interval: fleet.Duration{Duration: interval.value},
			},
		}, nil
	}

	lockCalled := make(chan struct{}, 1)
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		select {
		case lockCalled <- struct{}{}:
		default:
			// OK
		}
		return true, nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)

	webhooksLogger := log.NewNopLogger()
	webhooksInterval := appConfig.WebhookSettings.Interval.ValueOr(30 * time.Second)
	webhooks, err := New(ctx, "webhooks", "test_instance", webhooksInterval, ds, webhooksLogger)
	require.NoError(t, err)

	webhooks.setConfigInterval(200 * time.Millisecond)
	webhooks.SetConfigCheck(SetWebhooksConfigCheck(ctx, ds, webhooksLogger))
	webhooks.AddJob("cron_webhooks", func(ctx context.Context) (interface{}, error) {
		return DoWebhooks(ctx, ds, webhooksLogger, service.NewMemFailingPolicySet())
	}, func(interface{}, error) {})

	select {
	case <-configLoaded:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: initial config load")
	}

	interval.Lock()
	interval.value = 1 * time.Second
	interval.Unlock()

	select {
	case <-lockCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: schedInterval change did not trigger lock call")
	}
}
