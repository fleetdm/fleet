package service

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
	oc := &OrbitClient{}
	oc.configCache.config = cfg
	oc.configCache.lastUpdated = time.Now().Add(1 * time.Hour)
	return oc
}

func TestConfigReceiverCalls(t *testing.T) {
	var called1, called2 bool

	testmsg := json.RawMessage("testing")

	rfunc1 := fleet.OrbitReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		if !reflect.DeepEqual(cfg.Flags, testmsg) {
			return errors.New("not equal testmsg")
		}
		called1 = true
		return nil
	})
	rfunc2 := fleet.OrbitReceiverFunc(func(cfg *fleet.OrbitConfig) error {
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

	rfunc1 := fleet.OrbitReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		called1 = true
		return nil
	})
	rfunc2 := fleet.OrbitReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		called2 = true
		return nil
	})
	err1 := errors.New("error1")
	err2 := errors.New("error2")
	efunc1 := fleet.OrbitReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		return err1
	})
	efunc2 := fleet.OrbitReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		return err2
	})
	// Make sure we don't get stuck or crash on receiver panic
	pfunc := fleet.OrbitReceiverFunc(func(cfg *fleet.OrbitConfig) error {
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
