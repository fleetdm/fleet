package client

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	t.Run(
		"config cache", func(t *testing.T) {
			oc := OrbitClient{}
			oc.configCache.config = &fleet.OrbitConfig{}
			oc.configCache.lastUpdated = time.Now().Add(1 * time.Second)
			config, err := oc.GetConfig()
			require.NoError(t, err)
			require.Equal(t, oc.configCache.config, config)
		},
	)
	t.Run(
		"config cache error", func(t *testing.T) {
			oc := OrbitClient{}
			oc.configCache.config = nil
			oc.configCache.err = errors.New("test error")
			oc.configCache.lastUpdated = time.Now().Add(1 * time.Second)
			config, err := oc.GetConfig()
			require.Error(t, err)
			require.Equal(t, oc.configCache.config, config)
		},
	)
}

func clientWithConfig(cfg *fleet.OrbitConfig) *OrbitClient {
	ctx, cancel := context.WithCancel(context.Background())
	oc := &OrbitClient{
		receiverUpdateContext:    ctx,
		receiverUpdateCancelFunc: cancel,
	}
	oc.configCache.config = cfg
	oc.configCache.lastUpdated = time.Now().Add(1 * time.Hour)
	return oc
}

func TestConfigReceiverCalls(t *testing.T) {
	var called1, called2 bool

	testmsg := json.RawMessage("testing")

	rfunc1 := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		if !reflect.DeepEqual(cfg.Flags, testmsg) {
			return errors.New("not equal testmsg")
		}
		called1 = true
		return nil
	})
	rfunc2 := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		if !reflect.DeepEqual(cfg.Flags, testmsg) {
			return errors.New("not equal testmsg")
		}
		called2 = true
		return nil
	})

	client := clientWithConfig(&fleet.OrbitConfig{Flags: testmsg})
	client.RegisterConfigReceiver(rfunc1)
	client.RegisterConfigReceiver(rfunc2)

	err := client.RunConfigReceivers()
	require.NoError(t, err)

	require.True(t, called1)
	require.True(t, called2)
}

func TestConfigReceiverErrors(t *testing.T) {
	var called1, called2 bool

	rfunc1 := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		called1 = true
		return nil
	})
	rfunc2 := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		called2 = true
		return nil
	})
	err1 := errors.New("error1")
	err2 := errors.New("error2")
	efunc1 := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		return err1
	})
	efunc2 := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		return err2
	})
	// Make sure we don't get stuck or crash on receiver panic
	pfunc := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		panic("woah")
	})

	client := clientWithConfig(&fleet.OrbitConfig{})
	client.RegisterConfigReceiver(efunc1)
	client.RegisterConfigReceiver(rfunc1)
	client.RegisterConfigReceiver(efunc2)
	client.RegisterConfigReceiver(rfunc2)
	client.RegisterConfigReceiver(pfunc)

	err := client.RunConfigReceivers()
	require.ErrorIs(t, err, err1)
	require.ErrorIs(t, err, err2)

	require.True(t, called1)
	require.True(t, called2)
}

func TestExecuteConfigReceiversCancel(t *testing.T) {
	client := clientWithConfig(&fleet.OrbitConfig{})
	client.ReceiverUpdateInterval = 100 * time.Millisecond

	var calls1, calls2 int
	requiredCalls := 4

	cfunc := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		calls1++
		if calls1 == requiredCalls {
			client.receiverUpdateCancelFunc()
		}
		return nil
	})

	rfunc := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		calls2++
		return nil
	})

	client.RegisterConfigReceiver(cfunc)
	client.RegisterConfigReceiver(rfunc)

	err := client.ExecuteConfigReceivers()

	require.Nil(t, err)
	require.Equal(t, requiredCalls, calls1)
	require.Equal(t, requiredCalls, calls2)
}

func TestExecuteConfigReceiversInterrupt(t *testing.T) {
	client := clientWithConfig(&fleet.OrbitConfig{})
	defer client.receiverUpdateCancelFunc()

	client.ReceiverUpdateInterval = 100 * time.Millisecond

	var called bool
	rfunc := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		called = true
		return nil
	})

	client.RegisterConfigReceiver(rfunc)

	finChan := make(chan error)
	go func() {
		finChan <- client.ExecuteConfigReceivers()
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		client.receiverUpdateCancelFunc()
	}()

	select {
	case err := <-finChan:
		require.Nil(t, err)
		require.True(t, called)
	case <-time.NewTimer(2 * time.Second).C:
		require.Fail(t, "receiver interrupt cancel didn't work")
	}
}

func TestExecuteConfigReceiversBackoffOnError(t *testing.T) {
	client := clientWithConfig(&fleet.OrbitConfig{})
	client.ReceiverUpdateInterval = 1 * time.Second

	var callTimes []time.Time
	callCount := 0
	// 3 failures then cancel: intervals should be ~1s (base tick), ~2s, ~4s.
	targetCalls := 4

	rfunc := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		callTimes = append(callTimes, time.Now())
		callCount++
		if callCount >= targetCalls {
			client.receiverUpdateCancelFunc()
			return nil
		}
		return errors.New("server error")
	})

	client.RegisterConfigReceiver(rfunc)

	done := make(chan error, 1)
	go func() { done <- client.ExecuteConfigReceivers() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("test timed out waiting for ExecuteConfigReceivers")
	}
	require.Equal(t, targetCalls, callCount)

	// Verify each successive interval is strictly longer than the previous.
	// Call 0->1 is the base tick (~1s), 1->2 should be ~2s, 2->3 should be ~4s.
	require.GreaterOrEqual(t, len(callTimes), 3, "need at least 3 calls to verify growth")
	for i := 1; i < len(callTimes)-1; i++ {
		prev := callTimes[i].Sub(callTimes[i-1])
		curr := callTimes[i+1].Sub(callTimes[i])
		assert.Greater(t, curr, prev,
			"interval %d->%d (%v) should be greater than %d->%d (%v)",
			i, i+1, curr, i-1, i, prev)
	}
}

func TestExecuteConfigReceiversResetOnSuccess(t *testing.T) {
	client := clientWithConfig(&fleet.OrbitConfig{})
	client.ReceiverUpdateInterval = 1 * time.Second

	callCount := 0
	var intervalAfterRecovery time.Duration
	var recoveryStart time.Time

	rfunc := fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		callCount++
		switch {
		case callCount <= 2:
			// First 2 calls fail -- build up backoff
			return errors.New("server error")
		case callCount == 3:
			// Success -- should reset backoff
			recoveryStart = time.Now()
			return nil
		case callCount == 4:
			// Next call should be at base interval (~1s), not backed off
			intervalAfterRecovery = time.Since(recoveryStart)
			client.receiverUpdateCancelFunc()
			return nil
		}
		return nil
	})

	client.RegisterConfigReceiver(rfunc)

	done := make(chan error, 1)
	go func() { done <- client.ExecuteConfigReceivers() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("test timed out waiting for ExecuteConfigReceivers")
	}
	require.Equal(t, 4, callCount)

	// After recovery, interval should be close to base (1s), not backed off.
	// Use 2s as the upper bound: base (1s) + jitter (up to 10%) + scheduling slack.
	assert.Less(t, intervalAfterRecovery, 2*time.Second,
		"after success, interval should reset near base, got %v", intervalAfterRecovery)
}
