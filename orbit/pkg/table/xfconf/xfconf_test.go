//go:build linux
// +build linux

package xfconf

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/kolide/kit/fsutil"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func Test_getUserConfig(t *testing.T) {
	t.Parallel()

	tmpDefaultDir, tmpUserDir := setUpConfigFiles(t)

	xfconf := xfconfTable{
		logger: log.NewNopLogger(),
	}

	testUsername := "testUser"

	// Get the default config without error
	defaultConfig, err := xfconf.getDefaultConfig()
	require.NoError(t, err, "expected no error fetching default xfconfig")
	require.Greater(t, len(defaultConfig), 0)
	// Confirm lock-screen-suspend-hibernate is false now so we can validate that it got overridden after
	powerManagerChannelConfig, ok := defaultConfig[filepath.Join(tmpDefaultDir, xfconfChannelXmlPath, "xfce4-power-manager.xml")]
	require.True(t, ok, "invalid default data format -- missing channel file")
	powerManagerChannel, ok := powerManagerChannelConfig["channel/xfce4-power-manager"]
	require.True(t, ok, "invalid default data format -- missing channel")
	powerManagerProperties, ok := powerManagerChannel.(map[string]interface{})["xfce4-power-manager"]
	require.True(t, ok, "invalid default data format -- missing xfce4-power-manager property")
	lockScreenSuspendHibernate, ok := powerManagerProperties.(map[string]interface{})["lock-screen-suspend-hibernate"]
	require.True(t, ok, "invalid default data format -- missing lock-screen-suspend-hibernate property")
	require.Equal(t, "false", lockScreenSuspendHibernate)

	// Get the combined config without error
	config, err := xfconf.generateForUser(&user.User{Username: testUsername}, table.QueryContext{}, defaultConfig)
	require.NoError(t, err, "expected no error fetching xfconf config")

	// Confirm we have some data in the config and that it looks correct
	require.Greater(t, len(config), 0)
	for _, configRow := range config {
		// Confirm username was set correctly on all rows
		require.Equalf(t, testUsername, configRow["username"], "unexpected username: %s", configRow["username"])

		// Confirm path was set correctly on all rows
		require.True(t, strings.HasPrefix(configRow["path"], filepath.Join(tmpDefaultDir, xfconfChannelXmlPath)) ||
			strings.HasPrefix(configRow["path"], filepath.Join(tmpUserDir, xfconfChannelXmlPath)),
			"unexpected path: %s", configRow["path"])

		// Confirm each row came from an expected channel
		require.Truef(t, (strings.HasPrefix(configRow["fullkey"], "channel/xfce4-session") ||
			strings.HasPrefix(configRow["fullkey"], "channel/xfce4-power-manager") ||
			strings.HasPrefix(configRow["fullkey"], "channel/thunar-volman")),
			"unexpected channel: %s", configRow["fullkey"])

		// Confirm that we took user-specific config values over default ones
		if configRow["fullkey"] == "channel/xfce4-power-manager/xfce4-power-manager/lock-screen-suspend-hibernate" {
			require.Equal(t, "true", configRow["value"], "default settings for power manager not overridden by user settings")
		}
	}

	// Query with a constraint this time
	constraintList := table.ConstraintList{
		Affinity: table.ColumnTypeText,
		Constraints: []table.Constraint{
			{
				Operator:   table.OperatorEquals,
				Expression: "*/autoopen*",
			},
		},
	}
	q := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"query": constraintList,
		},
	}
	constrainedConfig, err := xfconf.generateForUser(&user.User{Username: testUsername}, q, defaultConfig)
	require.NoError(t, err, "expected no error fetching xfconf config with query constraints")
	require.Equal(t, 1, len(constrainedConfig), "query wrong number of rows, expected exactly 1")
	require.Equal(t, "channel/thunar-volman/autoopen/enabled", constrainedConfig[0]["fullkey"], "query fetched wrong row")
	require.Equal(t, "false", constrainedConfig[0]["value"], "fetched incorrect value for autoopen enabled")

	// Confirm that if we run into an error (e.g. requested user not existing), we fail soft -- still
	// returning a config, no error.
	fakeUserConstraintList := table.ConstraintList{
		Affinity: table.ColumnTypeText,
		Constraints: []table.Constraint{
			{
				Operator:   table.OperatorEquals,
				Expression: "AFakeUserThatDoesNotExist",
			},
		},
	}
	fakeUserQuery := table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"username": fakeUserConstraintList,
		},
	}
	fakeUserNoConfig, err := xfconf.generate(context.TODO(), fakeUserQuery)
	require.NoError(t, err, "expected no error fetching xfconf config")
	require.Equal(t, 0, len(fakeUserNoConfig), "expected no rows")
}

func setUpConfigFiles(t *testing.T) (string, string) {
	// Make a temporary directory for default config, put config files there
	tmpDefaultDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDefaultDir, xfconfChannelXmlPath), 0755), "error making temp directory")
	fsutil.CopyFile(filepath.Join("testdata", "xfce4-session.xml"), filepath.Join(tmpDefaultDir, xfconfChannelXmlPath, "xfce4-session.xml"))
	fsutil.CopyFile(filepath.Join("testdata", "xfce4-power-manager-default.xml"), filepath.Join(tmpDefaultDir, xfconfChannelXmlPath, "xfce4-power-manager.xml"))

	// Set the environment variable for the default directory
	os.Setenv("XDG_CONFIG_DIRS", tmpDefaultDir)

	// Make a temporary directory for user-specific config, put config files there
	tmpUserDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpUserDir, xfconfChannelXmlPath), 0755), "error making temp directory")
	fsutil.CopyFile(filepath.Join("testdata", "xfce4-power-manager.xml"), filepath.Join(tmpUserDir, xfconfChannelXmlPath, "xfce4-power-manager.xml"))
	fsutil.CopyFile(filepath.Join("testdata", "thunar-volman.xml"), filepath.Join(tmpUserDir, xfconfChannelXmlPath, "thunar-volman.xml"))

	// Set the environment variable for the user config directory
	os.Setenv("XDG_CONFIG_HOME", tmpUserDir)

	return tmpDefaultDir, tmpUserDir
}
