package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
)

func TestEarlySessionCheck(t *testing.T) {
	_, ds := runServerWithMockedDS(t)
	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
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
	err := os.WriteFile(configPath, []byte(config), configFilePerms)
	require.NoError(t, err)

	_, err = runAppNoChecks([]string{"get", "queries", "--config", configPath})
	require.ErrorIs(t, err, service.ErrUnauthenticated)
}
