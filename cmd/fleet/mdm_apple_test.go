package main

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// validPrivateKey is a 32-byte string used to satisfy the server private-key
// length requirement in reconciliation tests.
var validPrivateKey = strings.Repeat("x", 32)

// assetsPresentFunc returns a GetAllMDMConfigAssetsByNameFunc that reports the
// requested assets as already stored.
func assetsPresentFunc() mock.GetAllMDMConfigAssetsByNameFunc {
	return func(ctx context.Context, n []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
	}
}

// notFoundErr is a minimal error that satisfies fleet.IsNotFound.
type notFoundErr struct{}

func (notFoundErr) Error() string    { return "not found" }
func (notFoundErr) IsNotFound() bool { return true }

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestInitAppleMDMPushService_DevModeReturnsNopPusher(t *testing.T) {
	// SetOverride enables dev mode and registers cleanup via t.
	dev_mode.SetOverride("FLEET_DEV_MDM_APPLE_DISABLE_PUSH", "1", t)

	pusher := initAppleMDMPushService(nil, discardLogger())
	require.IsType(t, nopPusher{}, pusher)
}

func TestCheckMDMAssetsExist(t *testing.T) {
	names := []fleet.MDMAssetName{fleet.MDMAssetAPNSCert, fleet.MDMAssetAPNSKey}

	t.Run("assets present returns true", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, n []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
		}
		found, err := checkMDMAssetsExist(context.Background(), ds, names)
		require.NoError(t, err)
		require.True(t, found)
	})

	t.Run("not found returns false without error", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, n []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return nil, notFoundErr{}
		}
		found, err := checkMDMAssetsExist(context.Background(), ds, names)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("partial result returns false without error", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, n []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return nil, mysql.ErrPartialResult
		}
		found, err := checkMDMAssetsExist(context.Background(), ds, names)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("other error is surfaced", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, n []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return nil, sql.ErrConnDone
		}
		found, err := checkMDMAssetsExist(context.Background(), ds, names)
		require.Error(t, err)
		require.False(t, found)
	})
}

func TestReconcileAppleMDMAPNsAndSCEPAssets(t *testing.T) {
	t.Run("neither APNs nor SCEP set is a no-op", func(t *testing.T) {
		ds := new(mock.Store)
		called := false
		reconcileAppleMDMAPNsAndSCEPAssets(context.Background(), config.FleetConfig{}, ds, discardLogger(),
			func(err error, msg string) { called = true })
		require.False(t, called, "initFatal must not be called when neither is configured")
		require.False(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked, "datastore must not be touched")
	})

	t.Run("configured without server private key fails fast", func(t *testing.T) {
		ds := new(mock.Store)
		cfg := config.FleetConfig{
			MDM: config.MDMConfig{AppleAPNsCert: "apns.cert"},
		}
		called := false
		reconcileAppleMDMAPNsAndSCEPAssets(context.Background(), cfg, ds, discardLogger(),
			func(err error, msg string) { called = true })
		require.True(t, called, "initFatal must be called when private key is missing")
		require.False(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked, "must fail before touching datastore")
	})

	t.Run("assets already present skips insert", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetAllMDMConfigAssetsByNameFunc = assetsPresentFunc()
		cfg := config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: validPrivateKey},
			MDM:    config.MDMConfig{AppleAPNsCert: "apns.cert", AppleSCEPCert: "scep.cert"},
		}
		called := false
		reconcileAppleMDMAPNsAndSCEPAssets(context.Background(), cfg, ds, discardLogger(),
			func(err error, msg string) { called = true })
		require.False(t, called)
		require.True(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked, "datastore is checked")
		require.False(t, ds.InsertMDMConfigAssetsFuncInvoked, "insert is skipped when assets exist")
	})
}

func TestReconcileAppleMDMABMAssets(t *testing.T) {
	t.Run("ABM not set is a no-op", func(t *testing.T) {
		ds := new(mock.Store)
		called := false
		reconcileAppleMDMABMAssets(context.Background(), config.FleetConfig{}, ds, discardLogger(),
			func(err error, msg string) { called = true })
		require.False(t, called)
		require.False(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
	})

	t.Run("ABM set without server private key fails fast", func(t *testing.T) {
		ds := new(mock.Store)
		cfg := config.FleetConfig{
			MDM: config.MDMConfig{AppleBMCert: "abm.cert"},
		}
		called := false
		reconcileAppleMDMABMAssets(context.Background(), cfg, ds, discardLogger(),
			func(err error, msg string) { called = true })
		require.True(t, called)
		require.False(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
	})
}
