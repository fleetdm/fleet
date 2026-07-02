package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConditionalAccessProxy is a minimal mock of ConditionalAccessMicrosoftProxy
// for the ticker cleanup test.
type testConditionalAccessProxy struct{}

func (m *testConditionalAccessProxy) Create(_ context.Context, _ string) (*conditional_access_microsoft_proxy.CreateResponse, error) {
	return nil, nil
}

func (m *testConditionalAccessProxy) Get(_ context.Context, _ string, _ string) (*conditional_access_microsoft_proxy.GetResponse, error) {
	return nil, nil
}

func (m *testConditionalAccessProxy) Delete(_ context.Context, _ string, _ string) (*conditional_access_microsoft_proxy.DeleteResponse, error) {
	return nil, nil
}

func (m *testConditionalAccessProxy) SetComplianceStatus(
	_ context.Context,
	_ string, _ string,
	_ string,
	_ string,
	_ bool,
	_, _, _ string,
	_ bool,
	_ time.Time,
) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
	return &conditional_access_microsoft_proxy.SetComplianceStatusResponse{
		MessageID: "test-message-id",
	}, nil
}

func (m *testConditionalAccessProxy) GetMessageStatus(
	_ context.Context, _ string, _ string, _ string,
) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error) {
	return &conditional_access_microsoft_proxy.GetMessageStatusResponse{
		Status: conditional_access_microsoft_proxy.MessageStatusCompleted,
	}, nil
}

// TestSetHostConditionalAccess_TickerCleanup verifies that the ticker used in
// setHostConditionalAccess for polling macOS message status is properly stopped
// after each call, preventing resource leaks.
//
// With the old code (`for range time.Tick(...)`) each call leaked a ticker
// because time.Tick never releases the underlying ticker. The fix uses the
// injectable newConditionalAccessTicker factory with a proper stop function.
func TestSetHostConditionalAccess_TickerCleanup(t *testing.T) {
	ds := new(mock.Store)

	ds.ConditionalAccessMicrosoftGetFunc = func(_ context.Context) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
		return &fleet.ConditionalAccessMicrosoftIntegration{
			TenantID:          "test-tenant",
			ProxyServerSecret: "test-secret",
			SetupDone:         true,
		}, nil
	}

	ds.SetHostConditionalAccessStatusFunc = func(_ context.Context, _ uint, _ bool, _ bool) error {
		return nil
	}

	proxy := &testConditionalAccessProxy{}

	svc, _ := newTestService(t, ds, nil, nil, &TestServerOpts{
		ConditionalAccessMicrosoftProxy: proxy,
	})

	// Unwrap the validationMiddleware to get the concrete *Service.
	concreteSvc := svc.(validationMiddleware).Service.(*Service)

	// Use a very short poll interval so the test runs fast.
	origWait := conditionalAccessSetWaitTime
	conditionalAccessSetWaitTime = 1 * time.Millisecond
	t.Cleanup(func() {
		conditionalAccessSetWaitTime = origWait
	})

	// Instrument the ticker factory to track Stop() calls.
	var tickersCreated atomic.Int32
	var tickersStopped atomic.Int32

	origTickerFactory := newConditionalAccessTicker
	newConditionalAccessTicker = func(d time.Duration) (<-chan time.Time, func()) {
		tickersCreated.Add(1)
		ticker := time.NewTicker(d)
		return ticker.C, func() {
			tickersStopped.Add(1)
			ticker.Stop()
		}
	}
	t.Cleanup(func() {
		newConditionalAccessTicker = origTickerFactory
	})

	hostCA := &fleet.HostConditionalAccessStatus{
		HostID:            1,
		DeviceID:          "device-1",
		UserPrincipalName: "user@example.com",
		DisplayName:       "Test Host",
		OSVersion:         "14.0",
	}

	const iterations = 10
	for i := 0; i < iterations; i++ {
		err := concreteSvc.setHostConditionalAccess(
			uint(i+1), // hostID
			"darwin",  // platform -- triggers the ticker polling path
			hostCA,
			true, // managed
			true, // compliant
			nil,  // failingPolicyIDs
		)
		require.NoError(t, err)
	}

	created := tickersCreated.Load()
	stopped := tickersStopped.Load()
	t.Logf("tickers created: %d, stopped: %d", created, stopped)

	// Each darwin call should create exactly one ticker.
	assert.Equal(t, int32(iterations), created, "expected %d tickers to be created", iterations)
	// With the fix, every created ticker must be stopped.
	// With the old time.Tick code, Stop() is never called, so stopped would be 0.
	assert.Equal(t, created, stopped, "every created ticker must be stopped to avoid resource leaks")
}
