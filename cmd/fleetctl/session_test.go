package main

import (
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestEarlySessionCheck(t *testing.T) {
	_, ds := runServerWithMockedDS(t)
	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, error) {
		return nil, nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return nil, errors.New("invalid session")
	}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	config := `contexts:
  default:
    tls-skip-verify: true
    token: phIEGWGzKxXui1uZYFBXFwZ1Wv1iMxl79gbqMbOmMxgyZP2O5jga5qyhvEjzlGsdM7ax93iDqjnVSu9Fi8q1/w==`
	err := ioutil.WriteFile(configPath, []byte(config), configFilePerms)
	require.NoError(t, err)
	_, exitErr, err := runAppNoChecks([]string{"get", "queries", "--config", configPath})
	require.Error(t, err)
	require.NotNil(t, exitErr)
	require.True(t, errors.Is(err, invalidSessionErr))
}
