package cached_mysql

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClone(t *testing.T) {
	var nilRawMessage *json.RawMessage

	tests := []struct {
		name string
		src  interface{}
		want interface{}
	}{
		{
			name: "string",
			src:  "foo",
			want: "foo",
		},
		{
			name: "struct",
			src: fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					EnableAnalytics: true,
				},
			},
			want: fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					EnableAnalytics: true,
				},
			},
		},
		{
			name: "pointer to struct",
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
			name: "slice",
			src:  []string{"foo", "bar"},
			want: []string{"foo", "bar"},
		},
		{
			name: "pointer to slice",
			src:  &[]string{"foo", "bar"},
			want: &[]string{"foo", "bar"},
		},
		{
			name: "nil",
			src:  nil,
			want: nil,
		},
		{
			name: "nil pointer",
			src:  nilRawMessage,
			want: nil,
		},
		{
			name: "pointer to struct with nested slice",
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
			clone, err := clone(tc.src)
			require.NoError(t, err)
			assert.Equal(t, tc.want, clone)

			v1, v2 := reflect.ValueOf(tc.src), reflect.ValueOf(clone)
			if k := v1.Kind(); k == reflect.Pointer || k == reflect.Slice || k == reflect.Map || k == reflect.Chan || k == reflect.Func || k == reflect.UnsafePointer {
				if clone == nil {
					assert.True(t, v1.IsNil())
					return
				}
				require.Equal(t, v1.Kind(), v2.Kind())
				assert.NotEqual(t, v1.Pointer(), v2.Pointer())
			}

			// ensure that writing to src does not alter the cloned value (i.e. that
			// the nested fields are deeply cloned too).
			switch src := tc.src.(type) {
			case []string:
				if len(src) > 0 {
					src[0] = "modified"
					assert.NotEqual(t, src, clone)
				}
			case *[]string:
				if len(*src) > 0 {
					(*src)[0] = "modified"
					assert.NotEqual(t, src, clone)
				}
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

	scheduledQueries, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, dbScheduledQueries, scheduledQueries)

	// change "stored" dbScheduledQueries.
	dbScheduledQueries = fleet.ScheduledQueryList{
		{
			ID:   3,
			Name: "test-schedule-3",
		},
	}

	scheduledQueries2, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, scheduledQueries2, scheduledQueries) // returns the new db entry
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

	options, err := ds.TeamAgentOptions(context.Background(), 1)
	require.NoError(t, err)
	require.JSONEq(t, string(testOptions), string(*options))

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

	options, err = ds.TeamAgentOptions(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.JSONEq(t, string(updateOptions), string(*options))

	// deleting a team removes the agent options from the cache
	err = ds.DeleteTeam(context.Background(), testTeam.ID)
	require.NoError(t, err)

	_, err = ds.TeamAgentOptions(context.Background(), testTeam.ID)
	require.Error(t, err)
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
	require.True(t, mockedDS.TeamFeaturesFuncInvoked)
	mockedDS.TeamFeaturesFuncInvoked = false

	// get it again, will retrieve it from the cache
	features, err = ds.TeamFeatures(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testFeatures, *features)
	require.False(t, mockedDS.TeamFeaturesFuncInvoked)

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

	features, err = ds.TeamFeatures(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.Equal(t, updateFeatures, *features)
	require.False(t, mockedDS.TeamFeaturesFuncInvoked)

	// deleting a team removes the features from the cache
	err = ds.DeleteTeam(context.Background(), testTeam.ID)
	require.NoError(t, err)

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
			CustomSettings:                 []string{"a", "b"},
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
	require.True(t, mockedDS.TeamMDMConfigFuncInvoked)
	mockedDS.TeamMDMConfigFuncInvoked = false

	// get it again, will get it from cache
	mdmConfig, err = ds.TeamMDMConfig(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, testMDMConfig, *mdmConfig)
	require.False(t, mockedDS.TeamMDMConfigFuncInvoked)

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

	mdmConfig, err = ds.TeamMDMConfig(context.Background(), testTeam.ID)
	require.NoError(t, err)
	require.Equal(t, updateMDMConfig, *mdmConfig)
	require.False(t, mockedDS.TeamMDMConfigFuncInvoked)

	// deleting a team removes the config from the cache
	err = ds.DeleteTeam(context.Background(), testTeam.ID)
	require.NoError(t, err)

	_, err = ds.TeamMDMConfig(context.Background(), testTeam.ID)
	require.Error(t, err)
	require.True(t, mockedDS.TeamMDMConfigFuncInvoked)
}
