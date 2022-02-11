package cached_mysql

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachedAppConfig(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS)

	var appConfigSet *fleet.AppConfig
	mockedDS.NewAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
		appConfigSet = info
		return info, nil
	}
	mockedDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfigSet, nil
	}
	mockedDS.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
		appConfigSet = info
		return nil
	}
	_, err := ds.NewAppConfig(context.Background(), &fleet.AppConfig{
		HostSettings: fleet.HostSettings{
			AdditionalQueries: ptr.RawMessage(json.RawMessage(`"TestCachedAppConfig"`)),
		},
	})
	require.NoError(t, err)

	t.Run("NewAppConfig", func(t *testing.T) {
		data, err := ds.AppConfig(context.Background())
		require.NoError(t, err)

		require.NotEmpty(t, data)
		assert.Equal(t, json.RawMessage(`"TestCachedAppConfig"`), *data.HostSettings.AdditionalQueries)
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

		assert.True(t, mockedDS.SaveAppConfigFuncInvoked)

		ac, err := ds.AppConfig(context.Background())
		require.NoError(t, err)
		require.NotNil(t, ac.HostSettings.AdditionalQueries)
		assert.Equal(t, json.RawMessage(`"NewSAVED"`), *ac.HostSettings.AdditionalQueries)
	})

	t.Run("External SaveAppConfig gets caught", func(t *testing.T) {
		mockedDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				HostSettings: fleet.HostSettings{
					AdditionalQueries: ptr.RawMessage(json.RawMessage(`"SavedSomewhereElse"`)),
				},
			}, nil
		}

		time.Sleep(2 * time.Second)

		ac, err := ds.AppConfig(context.Background())
		require.NoError(t, err)
		require.NotNil(t, ac.HostSettings.AdditionalQueries)
		assert.Equal(t, json.RawMessage(`"SavedSomewhereElse"`), *ac.HostSettings.AdditionalQueries)
	})
}

func TestCachedPacksforHost(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithPacksExpiration(100*time.Millisecond))

	dbPacks := []*fleet.Pack{
		{
			ID:   1,
			Name: "test-pack-1",
		},
		{
			ID:   2,
			Name: "test-pack-2",
		},
	}
	called := 0
	mockedDS.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*fleet.Pack, err error) {
		called++
		return dbPacks, nil
	}

	packs, err := ds.ListPacksForHost(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbPacks, packs)

	// change "stored" dbPacks.
	dbPacks = []*fleet.Pack{
		{
			ID:   1,
			Name: "test-pack-1",
		},
		{
			ID:   3,
			Name: "test-pack-3",
		},
	}

	packs2, err := ds.ListPacksForHost(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, packs, packs2) // returns the old cached value
	require.Equal(t, 1, called)

	time.Sleep(200 * time.Millisecond)

	packs3, err := ds.ListPacksForHost(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbPacks, packs3) // returns the old cached value
	require.Equal(t, 2, called)
}

func TestCachedListScheduledQueriesInPack(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithScheduledQueriesExpiration(100*time.Millisecond))

	dbScheduledQueries := []*fleet.ScheduledQuery{
		{
			ID:   1,
			Name: "test-schedule-1",
		},
		{
			ID:   2,
			Name: "test-schedule-2",
		},
	}
	called := 0
	mockedDS.ListScheduledQueriesInPackFunc = func(ctx context.Context, packID uint) ([]*fleet.ScheduledQuery, error) {
		called++
		return dbScheduledQueries, nil
	}

	scheduledQueries, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbScheduledQueries, scheduledQueries)

	// change "stored" dbScheduledQueries.
	dbScheduledQueries = []*fleet.ScheduledQuery{
		{
			ID:   3,
			Name: "test-schedule-3",
		},
	}

	scheduledQueries2, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, scheduledQueries, scheduledQueries2) // returns the new db entry
	require.Equal(t, 1, called)

	time.Sleep(200 * time.Millisecond)

	scheduledQueries3, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbScheduledQueries, scheduledQueries3) // returns the new db entry
	require.Equal(t, 2, called)
}

func TestCachedTeamAgentOptions(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithTeamAgentOptionsExpiration(100*time.Millisecond))

	testOptions := json.RawMessage(`
{
  "config": {
    "options": {
      "logger_plugin": "tls",
      "pack_delimiter": "/",
      "logger_tls_period": 10,
      "distributed_plugin": "tls",
      "disable_distributed": false,
      "logger_tls_endpoint": "/api/v1/osquery/log",
      "distributed_interval": 10,
      "distributed_tls_max_attempts": 3
    },
    "decorators": {
      "load": [
        "SELECT uuid AS host_uuid FROM system_info;",
        "SELECT hostname AS hostname FROM system_info;"
      ]
    }
  },
  "overrides": {}
}
`)

	testTeam := &fleet.Team{
		ID:           1,
		CreatedAt:    time.Now(),
		Name:         "test",
		AgentOptions: &testOptions,
	}

	deleted := false
	mockedDS.TeamAgentOptionsFunc = func(ctx context.Context, teamID uint) (*json.RawMessage, error) {
		if deleted {
			return nil, errors.New("not found")
		}
		return &testOptions, nil
	}
	mockedDS.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}
	mockedDS.DeleteTeamFunc = func(ctx context.Context, teamID uint) error {
		deleted = true
		return nil
	}

	options, err := ds.TeamAgentOptions(context.Background(), 1)
	require.NoError(t, err)
	require.JSONEq(t, string(testOptions), string(*options))

	// saving a team updates agent options in cache
	updateOptions := json.RawMessage(`
{}
`)
	updateTeam := &fleet.Team{
		ID:           testTeam.ID,
		CreatedAt:    testTeam.CreatedAt,
		Name:         testTeam.Name,
		AgentOptions: &updateOptions,
	}

	_, err = ds.SaveTeam(context.Background(), updateTeam)
	require.NoError(t, err)

	options, err = ds.TeamAgentOptions(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.JSONEq(t, string(updateOptions), string(*options))

	// deleting a team removes the agent options from the cache
	err = ds.DeleteTeam(context.Background(), testTeam.ID)
	require.NoError(t, err)

	_, err = ds.TeamAgentOptions(context.Background(), testTeam.ID)
	require.Error(t, err)
}
