package main

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRedisConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cfg     config.RedisConfig
		wantErr bool
		wantSub string
	}{
		{
			name: "host cache disabled with zero ttl is ok",
			cfg:  config.RedisConfig{HostCacheEnabled: false, HostCacheTTL: 0},
		},
		{
			name: "host cache disabled with negative ttl is ok",
			cfg:  config.RedisConfig{HostCacheEnabled: false, HostCacheTTL: -1 * time.Second},
		},
		{
			name: "host cache enabled with positive ttl is ok",
			cfg:  config.RedisConfig{HostCacheEnabled: true, HostCacheTTL: 5 * time.Minute},
		},
		{
			name:    "host cache enabled with zero ttl is rejected",
			cfg:     config.RedisConfig{HostCacheEnabled: true, HostCacheTTL: 0},
			wantErr: true,
			wantSub: "host_cache_ttl must be > 0",
		},
		{
			name:    "host cache enabled with negative ttl is rejected",
			cfg:     config.RedisConfig{HostCacheEnabled: true, HostCacheTTL: -1 * time.Second},
			wantErr: true,
			wantSub: "host_cache_ttl must be > 0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedisConfig(tc.cfg)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantSub)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestBuildRedisPoolConfigStripsScheme(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "scheme stripped", input: "redis://example.com:6379", want: "example.com:6379"},
		{name: "no scheme passes through", input: "example.com:6379", want: "example.com:6379"},
		{name: "empty passes through", input: "", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRedisPoolConfig(config.RedisConfig{Address: tc.input})
			assert.Equal(t, tc.want, got.Server)
		})
	}
}
