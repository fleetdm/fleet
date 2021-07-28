package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
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
		return &fleet.AppConfig{EnableAnalytics: true}, nil
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
		return &fleet.AppConfig{EnableAnalytics: true}, nil
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
		return &fleet.AppConfig{EnableAnalytics: false}, nil
	}

	err := trySendStatistics(ds, fleet.StatisticsFrequency, ts.URL)
	require.NoError(t, err)
	assert.False(t, called)
}
