package cached_mysql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/datastoretest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachedAppConfig(t *testing.T) {
	pool := datastoretest.SetupRedisForTest(t, false, false)
	conn := pool.Get()
	data, err := redigo.Bytes(conn.Do("GET", CacheKeyAppConfig))
	require.Equal(t, redigo.ErrNil, err)

	mockedDS := new(mock.Store)
	ds := New(mockedDS, pool)

	var appConfigSet *fleet.AppConfig
	mockedDS.NewAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
		appConfigSet = info
		return info, nil
	}
	mockedDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfigSet, err
	}
	mockedDS.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
		appConfigSet = info
		return nil
	}
	_, err = ds.NewAppConfig(context.Background(), &fleet.AppConfig{
		HostSettings: fleet.HostSettings{
			AdditionalQueries: ptr.RawMessage(json.RawMessage(`"TestCachedAppConfig"`)),
		},
	})
	require.NoError(t, err)

	t.Run("NewAppConfig", func(t *testing.T) {
		data, err = redigo.Bytes(conn.Do("GET", CacheKeyAppConfig))
		require.NoError(t, err)

		require.NotEmpty(t, data)
		newAc := &fleet.AppConfig{}
		require.NoError(t, json.Unmarshal(data, &newAc))
		require.NotNil(t, newAc.HostSettings.AdditionalQueries)
		assert.Equal(t, json.RawMessage(`"TestCachedAppConfig"`), *newAc.HostSettings.AdditionalQueries)
	})

	t.Run("AppConfig", func(t *testing.T) {
		require.False(t, mockedDS.AppConfigFuncInvoked)
		ac, err := ds.AppConfig(context.Background())
		require.NoError(t, err)
		require.False(t, mockedDS.AppConfigFuncInvoked)

		require.Equal(t, ptr.RawMessage(json.RawMessage(`"TestCachedAppConfig"`)), ac.HostSettings.AdditionalQueries)
	})

	t.Run("SaveAppConfig", func(t *testing.T) {
		require.NoError(t, ds.SaveAppConfig(context.Background(), &fleet.AppConfig{
			HostSettings: fleet.HostSettings{
				AdditionalQueries: ptr.RawMessage(json.RawMessage(`"NewSAVED"`)),
			},
		}))

		data, err = redigo.Bytes(conn.Do("GET", CacheKeyAppConfig))
		require.NoError(t, err)

		require.NotEmpty(t, data)
		newAc := &fleet.AppConfig{}
		require.NoError(t, json.Unmarshal(data, &newAc))
		require.NotNil(t, newAc.HostSettings.AdditionalQueries)
		assert.Equal(t, json.RawMessage(`"NewSAVED"`), *newAc.HostSettings.AdditionalQueries)
	})
}
