package service

import (
	"errors"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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
