package cached_mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nilCloner struct{}

func (n *nilCloner) Clone() (fleet.Cloner, error) {
	var nn *nilCloner
	return nn, nil
}

func TestClone(t *testing.T) {
	tests := []struct {
		name string
		src  fleet.Cloner
		want fleet.Cloner
	}{
		{
			name: "appconfig",
			src: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					EnableAnalytics: true,
				},
			},
			want: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					EnableAnalytics: true,
				},
			},
		},
		{
			name: "nil",
			src:  (*nilCloner)(nil),
			want: (*nilCloner)(nil),
		},
		{
			name: "appconfig with nested slice",
			src: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					DebugHostIDs: []uint{1, 2, 3},
				},
			},
			want: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					DebugHostIDs: []uint{1, 2, 3},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clone, err := tc.src.Clone()
			require.NoError(t, err)
			assert.Equal(t, tc.want, clone)

			// ensure that writing to src does not alter the cloned value (i.e. that
			// the nested fields are deeply cloned too).
			switch src := tc.src.(type) {
			case *fleet.AppConfig:
				if len(src.ServerSettings.DebugHostIDs) > 0 {
					src.ServerSettings.DebugHostIDs[0] = 999
					assert.NotEqual(t, src.ServerSettings.DebugHostIDs, clone.(*fleet.AppConfig).ServerSettings.DebugHostIDs)
				}
			}
		})
	}
}

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
		Features: fleet.Features{
			AdditionalQueries: ptr.RawMessage(json.RawMessage(`"TestCachedAppConfig"`)),
		},
	})
	require.NoError(t, err)

	t.Run("NewAppConfig", func(t *testing.T) {
		data, err := ds.AppConfig(context.Background())
		require.NoError(t, err)

		require.NotEmpty(t, data)
		assert.Equal(t, json.RawMessage(`"TestCachedAppConfig"`), *data.Features.AdditionalQueries)
	})

	t.Run("AppConfig", func(t *testing.T) {
		require.False(t, mockedDS.AppConfigFuncInvoked)
		ac, err := ds.AppConfig(context.Background())
		require.NoError(t, err)
		require.False(t, mockedDS.AppConfigFuncInvoked)

		require.Equal(t, ptr.RawMessage(json.RawMessage(`"TestCachedAppConfig"`)), ac.Features.AdditionalQueries)
	})

	t.Run("SaveAppConfig", func(t *testing.T) {
		require.NoError(t, ds.SaveAppConfig(context.Background(), &fleet.AppConfig{
			Features: fleet.Features{
				AdditionalQueries: ptr.RawMessage(json.RawMessage(`"NewSAVED"`)),
			},
		}))

		assert.True(t, mockedDS.SaveAppConfigFuncInvoked)

		ac, err := ds.AppConfig(context.Background())
		require.NoError(t, err)
		require.NotNil(t, ac.Features.AdditionalQueries)
		assert.Equal(t, json.RawMessage(`"NewSAVED"`), *ac.Features.AdditionalQueries)
	})

	t.Run("External SaveAppConfig gets caught", func(t *testing.T) {
		mockedDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				Features: fleet.Features{
					AdditionalQueries: ptr.RawMessage(json.RawMessage(`"SavedSomewhereElse"`)),
				},
			}, nil
		}

		time.Sleep(2 * time.Second)

		ac, err := ds.AppConfig(context.Background())
		require.NoError(t, err)
		require.NotNil(t, ac.Features.AdditionalQueries)
		assert.Equal(t, json.RawMessage(`"SavedSomewhereElse"`), *ac.Features.AdditionalQueries)
	})
}

func TestBypassAppConfig(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithAppConfigExpiration(time.Minute))

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

	// calling NewAppConfig initializes the cache
	_, err := ds.NewAppConfig(context.Background(), &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName: "A",
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// used the cached value
	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "A", ac.OrgInfo.OrgName)
	require.False(t, mockedDS.AppConfigFuncInvoked)

	// update and save it, calls the DB
	ac.OrgInfo.OrgName = "B"
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)
	require.True(t, mockedDS.SaveAppConfigFuncInvoked)
	mockedDS.SaveAppConfigFuncInvoked = false

	// read it back, uses the cache
	ac, err = ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "B", ac.OrgInfo.OrgName)
	require.False(t, mockedDS.AppConfigFuncInvoked)

	// simulate a database change from another process, not via the cached_mysql store
	ac.OrgInfo.OrgName = "C"
	err = mockedDS.SaveAppConfig(ctx, ac)
	require.NoError(t, err)

	// reading it via the store uses the old cached value
	ac, err = ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "B", ac.OrgInfo.OrgName)
	require.False(t, mockedDS.AppConfigFuncInvoked)

	// force-bypassing the cache gets the updated value
	ctx = ctxdb.BypassCachedMysql(ctx, true)
	ac, err = ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "C", ac.OrgInfo.OrgName)
	require.True(t, mockedDS.AppConfigFuncInvoked)
	mockedDS.AppConfigFuncInvoked = false

	// bypassing the cache to read AppConfig did update the cache, so if we don't
	// bypass it anymore, it now gets the updated value from the cache
	ctx = ctxdb.BypassCachedMysql(ctx, false)
	ac, err = ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, "C", ac.OrgInfo.OrgName)
	require.False(t, mockedDS.AppConfigFuncInvoked)
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

	// first call gets the result from the DB
	packs, err := ds.ListPacksForHost(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbPacks, packs)
	require.Same(t, dbPacks[0], packs[0])
	require.Equal(t, 1, called)

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

	// this call gets it from the cache
	packs2, err := ds.ListPacksForHost(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, packs, packs2)         // returns the old cached value
	require.NotSame(t, packs[0], packs2[0]) // have been cloned
	require.Equal(t, 1, called)

	time.Sleep(200 * time.Millisecond)

	// this call gets it from the DB again since the cache expired
	packs3, err := ds.ListPacksForHost(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbPacks, packs3)
	require.Same(t, dbPacks[0], packs3[0])
	require.Equal(t, 2, called)
}

func TestCachedListScheduledQueriesInPack(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithScheduledQueriesExpiration(100*time.Millisecond))

	dbScheduledQueries := fleet.ScheduledQueryList{
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
	mockedDS.ListScheduledQueriesInPackFunc = func(ctx context.Context, packID uint) (fleet.ScheduledQueryList, error) {
		called++
		return dbScheduledQueries, nil
	}

	// this initial call gets the result from the DB
	scheduledQueries, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbScheduledQueries, scheduledQueries)
	require.Same(t, dbScheduledQueries[0], scheduledQueries[0])
	require.Equal(t, 1, called)

	// change "stored" dbScheduledQueries.
	dbScheduledQueries = fleet.ScheduledQueryList{
		{
			ID:   3,
			Name: "test-schedule-3",
		},
	}

	// this call gets it from the cache
	scheduledQueries2, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, scheduledQueries2, scheduledQueries)
	require.NotSame(t, scheduledQueries[0], scheduledQueries2[0]) // has been cloned
	require.Equal(t, 1, called)

	time.Sleep(200 * time.Millisecond)

	// this call gets it from the DB again, since the cache expired
	scheduledQueries3, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbScheduledQueries, scheduledQueries3)
	require.Same(t, dbScheduledQueries[0], scheduledQueries3[0])
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
      "logger_tls_endpoint": "/api/osquery/log",
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
		ID:        1,
		CreatedAt: time.Now(),
		Name:      "test",
		Config: fleet.TeamConfig{
			AgentOptions: &testOptions,
		},
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

	// initial call reads from the DB
	options, err := ds.TeamAgentOptions(context.Background(), 1)
	require.NoError(t, err)
	require.JSONEq(t, string(testOptions), string(*options))
	require.Same(t, &testOptions, options)
	require.True(t, mockedDS.TeamAgentOptionsFuncInvoked)
	mockedDS.TeamAgentOptionsFuncInvoked = false

	// subsequent call reads from the cache
	options, err = ds.TeamAgentOptions(context.Background(), 1)
	require.NoError(t, err)
	require.JSONEq(t, string(testOptions), string(*options))
	require.NotSame(t, &testOptions, options)
	require.False(t, mockedDS.TeamAgentOptionsFuncInvoked)

	// saving a team updates agent options in cache
	updateOptions := json.RawMessage(`
{}
`)
	updateTeam := &fleet.Team{
		ID:        testTeam.ID,
		CreatedAt: testTeam.CreatedAt,
		Name:      testTeam.Name,
		Config: fleet.TeamConfig{
			AgentOptions: &updateOptions,
		},
	}

	_, err = ds.SaveTeam(context.Background(), updateTeam)
	require.NoError(t, err)
	require.True(t, mockedDS.SaveTeamFuncInvoked)
	mockedDS.SaveTeamFuncInvoked = false

	// reading reads it from the cache with the updated data
	options, err = ds.TeamAgentOptions(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.JSONEq(t, string(updateOptions), string(*options))
	require.NotSame(t, &updateOptions, options)
	require.False(t, mockedDS.TeamAgentOptionsFuncInvoked)

	// deleting a team removes the agent options from the cache
	err = ds.DeleteTeam(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.True(t, mockedDS.DeleteTeamFuncInvoked)
	mockedDS.DeleteTeamFuncInvoked = false

	// reading hits the DB as the cached item was removed
	_, err = ds.TeamAgentOptions(context.Background(), testTeam.ID)
	require.Error(t, err)
	require.True(t, mockedDS.TeamAgentOptionsFuncInvoked)
}

func TestCachedTeamFeatures(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithTeamFeaturesExpiration(100*time.Millisecond))

	ao := json.RawMessage(`{}`)
	aq := json.RawMessage(`{"foo": "bar"}`)
	testFeatures := fleet.Features{
		EnableHostUsers:         false,
		EnableSoftwareInventory: true,
		AdditionalQueries:       &aq,
		DetailQueryOverrides:    map[string]*string{"a": ptr.String("A"), "b": ptr.String("B")},
	}

	testTeam := fleet.Team{
		ID:        1,
		CreatedAt: time.Now(),
		Name:      "test",
		Config: fleet.TeamConfig{
			Features:     testFeatures,
			AgentOptions: &ao,
		},
	}

	deleted := false
	mockedDS.TeamFeaturesFunc = func(ctx context.Context, teamID uint) (*fleet.Features, error) {
		if deleted {
			return nil, errors.New("not found")
		}
		return &testFeatures, nil
	}
	mockedDS.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}
	mockedDS.DeleteTeamFunc = func(ctx context.Context, teamID uint) error {
		deleted = true
		return nil
	}

	// get it the first time, it will populate the cache
	features, err := ds.TeamFeatures(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testFeatures, *features)
	require.Same(t, &testFeatures, features)
	require.True(t, mockedDS.TeamFeaturesFuncInvoked)
	mockedDS.TeamFeaturesFuncInvoked = false

	// get it again, will retrieve it from the cache
	features, err = ds.TeamFeatures(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testFeatures, *features)
	require.NotSame(t, &testFeatures, features)
	require.False(t, mockedDS.TeamFeaturesFuncInvoked)

	// changing e.g. the DetailQueryOverrides map doesn't affect the stored value
	ptrA := features.DetailQueryOverrides["a"]
	*ptrA = "AAA"
	features.DetailQueryOverrides["c"] = ptr.String("C")
	require.NotEqual(t, testFeatures.DetailQueryOverrides, features.DetailQueryOverrides)

	// saving a team updates features in cache
	aq = json.RawMessage(`{"bar": "baz"}`)
	updateFeatures := fleet.Features{
		EnableHostUsers:         true,
		EnableSoftwareInventory: false,
		AdditionalQueries:       &aq,
		DetailQueryOverrides:    map[string]*string{"c": ptr.String("C")},
	}
	updateTeam := &fleet.Team{
		ID:        testTeam.ID,
		CreatedAt: testTeam.CreatedAt,
		Name:      testTeam.Name,
		Config: fleet.TeamConfig{
			Features:     updateFeatures,
			AgentOptions: &ao,
		},
	}

	_, err = ds.SaveTeam(context.Background(), updateTeam)
	require.NoError(t, err)
	require.True(t, mockedDS.SaveTeamFuncInvoked)
	mockedDS.SaveTeamFuncInvoked = false

	// this call gets it from the cache and gets the updated value set by SaveTeam
	features, err = ds.TeamFeatures(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.Equal(t, updateFeatures, *features)
	require.NotSame(t, &updateFeatures, features)
	require.False(t, mockedDS.TeamFeaturesFuncInvoked)

	// deleting a team removes the features from the cache
	err = ds.DeleteTeam(context.Background(), testTeam.ID)
	require.NoError(t, err)

	// reading hits the DB as the cached item was removed
	_, err = ds.TeamFeatures(context.Background(), testTeam.ID)
	require.Error(t, err)
	require.True(t, mockedDS.TeamFeaturesFuncInvoked)
}

func TestCachedTeamMDMConfig(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithTeamMDMConfigExpiration(100*time.Millisecond))
	ao := json.RawMessage(`{}`)

	testMDMConfig := fleet.TeamMDM{
		EnableDiskEncryption: true,
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.SetString("10.10.10"),
			Deadline:       optjson.SetString("1992-03-01"),
		},
		MacOSSettings: fleet.MacOSSettings{
			CustomSettings:                 []fleet.MDMProfileSpec{{Path: "a"}, {Path: "b"}},
			DeprecatedEnableDiskEncryption: ptr.Bool(false),
		},
		MacOSSetup: fleet.MacOSSetup{
			BootstrapPackage: optjson.SetString("bootstrap"),
		},
	}

	testTeam := fleet.Team{
		ID:        1,
		CreatedAt: time.Now(),
		Name:      "test",
		Config: fleet.TeamConfig{
			MDM:          testMDMConfig,
			AgentOptions: &ao,
		},
	}

	deleted := false
	mockedDS.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
		if deleted {
			return nil, errors.New("not found")
		}
		return &testMDMConfig, nil
	}
	mockedDS.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}
	mockedDS.DeleteTeamFunc = func(ctx context.Context, teamID uint) error {
		deleted = true
		return nil
	}

	// get the team's config, will load it into cache
	mdmConfig, err := ds.TeamMDMConfig(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testMDMConfig, *mdmConfig)
	require.Same(t, &testMDMConfig, mdmConfig)
	require.True(t, mockedDS.TeamMDMConfigFuncInvoked)
	mockedDS.TeamMDMConfigFuncInvoked = false

	// get it again, will get it from cache
	mdmConfig, err = ds.TeamMDMConfig(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testMDMConfig, *mdmConfig)
	require.NotSame(t, &testMDMConfig, mdmConfig)
	require.False(t, mockedDS.TeamMDMConfigFuncInvoked)

	// changing some deep value doesn't affect the stored value
	mdmConfig.MacOSSettings.CustomSettings[0] = fleet.MDMProfileSpec{Path: "c"}
	require.NotEqual(t, testMDMConfig, *mdmConfig)

	// saving a team updates config in cache
	updateMDMConfig := fleet.TeamMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.SetString("13.13.13"),
			Deadline:       optjson.SetString("2022-03-01"),
		},
		MacOSSettings: fleet.MacOSSettings{
			CustomSettings:                 nil,
			DeprecatedEnableDiskEncryption: ptr.Bool(true),
		},
	}
	updateTeam := &fleet.Team{
		ID:        testTeam.ID,
		CreatedAt: testTeam.CreatedAt,
		Name:      testTeam.Name,
		Config: fleet.TeamConfig{
			MDM:          updateMDMConfig,
			AgentOptions: &ao,
		},
	}

	_, err = ds.SaveTeam(context.Background(), updateTeam)
	require.NoError(t, err)
	require.True(t, mockedDS.SaveTeamFuncInvoked)
	mockedDS.SaveTeamFuncInvoked = false

	// this call gets it from the cache, with the updated value set by SaveTeam
	mdmConfig, err = ds.TeamMDMConfig(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.Equal(t, updateMDMConfig, *mdmConfig)
	require.NotSame(t, &updateMDMConfig, mdmConfig)
	require.False(t, mockedDS.TeamMDMConfigFuncInvoked)

	// deleting a team removes the config from the cache
	err = ds.DeleteTeam(context.Background(), testTeam.ID)
	require.NoError(t, err)

	// reading hits the DB as the cached item was removed
	_, err = ds.TeamMDMConfig(context.Background(), testTeam.ID)
	require.Error(t, err)
	require.True(t, mockedDS.TeamMDMConfigFuncInvoked)
}

func TestCachedQueryByName(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithQueryByNameExpiration(100*time.Millisecond))

	testQuery := &fleet.Query{
		ID:     1,
		TeamID: ptr.Uint(1),
		Packs:  []fleet.Pack{{ID: 2, Name: "a"}},
	}
	mockedDS.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return testQuery, nil
	}

	// first call gets the result from the DB
	query1, err := ds.QueryByName(context.Background(), nil, "q")
	require.NoError(t, err)
	require.Equal(t, testQuery, query1)
	require.Same(t, testQuery, query1)
	require.True(t, mockedDS.QueryByNameFuncInvoked)
	mockedDS.QueryByNameFuncInvoked = false

	// change "stored" query.
	testQuery = &fleet.Query{
		ID:     1,
		TeamID: ptr.Uint(2),
		Packs:  []fleet.Pack{{ID: 3, Name: "b"}},
	}

	// this call gets it from the cache
	query2, err := ds.QueryByName(context.Background(), nil, "q")
	require.NoError(t, err)
	require.Equal(t, query1, query2)   // returns the old cached value
	require.NotSame(t, query1, query2) // have been cloned
	require.False(t, mockedDS.QueryByNameFuncInvoked)

	// a deep change doesn't alter the stored value
	query2.Packs[0].Name = "Z"
	require.NotEqual(t, query1, query2)

	time.Sleep(200 * time.Millisecond)

	// this call gets it from the DB again since the cache expired
	query3, err := ds.QueryByName(context.Background(), nil, "q")
	require.NoError(t, err)
	require.Equal(t, testQuery, query3)
	require.Same(t, testQuery, query3)
	require.True(t, mockedDS.QueryByNameFuncInvoked)
}

func TestCachedResultCountForQuery(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithQueryResultsCountExpiration(100*time.Millisecond))

	testCount := 1
	mockedDS.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return testCount, nil
	}

	// first call gets the result from the DB
	c1, err := ds.ResultCountForQuery(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testCount, c1)
	require.True(t, mockedDS.ResultCountForQueryFuncInvoked)
	mockedDS.ResultCountForQueryFuncInvoked = false

	// change "stored" count.
	testCount = 2

	// this call gets it from the cache
	c2, err := ds.ResultCountForQuery(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, c1, c2) // returns the old cached value
	require.False(t, mockedDS.ResultCountForQueryFuncInvoked)

	time.Sleep(200 * time.Millisecond)

	// this call gets it from the DB again since the cache expired
	c3, err := ds.ResultCountForQuery(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testCount, c3)
	require.True(t, mockedDS.ResultCountForQueryFuncInvoked)
}

func TestCachedInsertMDMConfigAssets(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS).(*cachedMysql)

	mockedDS.InsertMDMConfigAssetsFunc = func(ctx context.Context, assets []fleet.MDMConfigAsset) error {
		return nil
	}

	assets := []fleet.MDMConfigAsset{
		{Name: "asset1", Value: []byte("value1")},
		{Name: "asset2", Value: []byte("value2")},
	}

	err := ds.InsertMDMConfigAssets(context.Background(), assets)
	require.NoError(t, err)

	for _, a := range assets {
		_, found := ds.c.Get(context.Background(), fmt.Sprintf(mdmConfigAssetKey, a.Name))
		assert.False(t, found)
	}

	assert.True(t, mockedDS.InsertMDMConfigAssetsFuncInvoked)
}

func TestCachedGetAllMDMConfigAssetsByName(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS, WithMDMConfigAssetExpiration(100*time.Millisecond)).(*cachedMysql)

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		"asset1": {Name: "asset1", Value: []byte("value1")},
		"asset2": {Name: "asset2", Value: []byte("value2")},
	}

	mockedDS.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		result := make(map[fleet.MDMAssetName]fleet.MDMConfigAsset)
		for _, name := range assetNames {
			if asset, ok := assets[name]; ok {
				result[name] = asset
			}
		}
		return result, nil
	}

	// First call gets the result from the DB
	names := []fleet.MDMAssetName{"asset1", "asset2"}
	res, err := ds.GetAllMDMConfigAssetsByName(context.Background(), names)
	require.NoError(t, err)
	require.Equal(t, assets, res)
	for name := range res {
		_, found := ds.c.Get(context.Background(), fmt.Sprintf(mdmConfigAssetKey, name))
		assert.True(t, found)
	}
	assert.True(t, mockedDS.GetAllMDMConfigAssetsByNameFuncInvoked)
	mockedDS.GetAllMDMConfigAssetsByNameFuncInvoked = false

	// Change "stored" assets.
	assets["asset1"] = fleet.MDMConfigAsset{Name: "asset1", Value: []byte("newvalue1")}

	// This call gets it from the cache
	res, err = ds.GetAllMDMConfigAssetsByName(context.Background(), names)
	require.NoError(t, err)
	require.NotEqual(t, assets["asset1"], res["asset1"]) // returns the old cached value
	assert.False(t, mockedDS.GetAllMDMConfigAssetsByNameFuncInvoked)
}

func TestCachedDeleteMDMConfigAssetsByName(t *testing.T) {
	t.Parallel()

	mockedDS := new(mock.Store)
	ds := New(mockedDS).(*cachedMysql)

	mockedDS.DeleteMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName) error {
		return nil
	}

	// Insert assets into cache
	assets := []fleet.MDMConfigAsset{
		{Name: "asset1", Value: []byte("value1")},
		{Name: "asset2", Value: []byte("value2")},
	}
	for _, a := range assets {
		ds.c.Set(context.Background(), fmt.Sprintf(mdmConfigAssetKey, a.Name), a, ds.mdmConfigAssetExp)
	}

	// Delete the assets
	names := []fleet.MDMAssetName{"asset1", "asset2"}
	err := ds.DeleteMDMConfigAssetsByName(context.Background(), names)
	require.NoError(t, err)

	// Ensure assets are removed from cache
	for _, name := range names {
		_, found := ds.c.Get(context.Background(), fmt.Sprintf(mdmConfigAssetKey, name))
		assert.False(t, found)
	}
	assert.True(t, mockedDS.DeleteMDMConfigAssetsByNameFuncInvoked)
}
