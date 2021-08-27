package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaybeSendStatistics(t *testing.T) {
	ds := new(mock.Store)

	requestBody := ""

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(requestBodyBytes)
	}))
	defer ts.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{EnableAnalytics: true}}, nil
	}

	ds.ShouldSendStatisticsFunc = func(frequency time.Duration) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{
			AnonymousIdentifier: "ident",
			FleetVersion:        "1.2.3",
			NumHostsEnrolled:    999,
		}, true, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func() error {
		recorded = true
		return nil
	}

	err := trySendStatistics(ds, fleet.StatisticsFrequency, ts.URL)
	require.NoError(t, err)
	assert.True(t, recorded)
	assert.Equal(t, `{"anonymousIdentifier":"ident","fleetVersion":"1.2.3","numHostsEnrolled":999}`, requestBody)
}

func TestMaybeSendStatisticsSkipsSendingIfNotNeeded(t *testing.T) {
	ds := new(mock.Store)

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{EnableAnalytics: true}}, nil
	}

	ds.ShouldSendStatisticsFunc = func(frequency time.Duration) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{}, false, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func() error {
		recorded = true
		return nil
	}

	err := trySendStatistics(ds, fleet.StatisticsFrequency, ts.URL)
	require.NoError(t, err)
	assert.False(t, recorded)
	assert.False(t, called)
}

func TestMaybeSendStatisticsSkipsIfNotConfigured(t *testing.T) {
	ds := new(mock.Store)

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	err := trySendStatistics(ds, fleet.StatisticsFrequency, ts.URL)
	require.NoError(t, err)
	assert.False(t, called)
}

type alwaysLocker struct{}

func (m *alwaysLocker) Lock(name string, owner string, expiration time.Duration) (bool, error) {
	return true, nil
}
func (m *alwaysLocker) Unlock(name string, owner string) error {
	return nil
}

func TestCronWebhooks(t *testing.T) {
	ds := new(mock.Store)

	endpointCalled := int32(0)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&endpointCalled, 1)
	}))
	defer ts.Close()

	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
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

	calledOnce := make(chan struct{})
	calledTwice := make(chan struct{})
	ds.TotalAndUnseenHostsSinceFunc = func(daysCount int) (int, int, error) {
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

	go cronWebhooks(ctx, ds, kitlog.With(kitlog.NewNopLogger(), "cron", "webhooks"), &alwaysLocker{}, "1234")

	<-calledOnce
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(1), atomic.LoadInt32(&endpointCalled))
	<-calledTwice
	time.Sleep(1 * time.Second)
	assert.GreaterOrEqual(t, int32(2), atomic.LoadInt32(&endpointCalled))
}
