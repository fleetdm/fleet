package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestAutoUpdateFleetMaintainedApps(t *testing.T) {
	// Cached versions for the title, semver-sorted newest-first (as the real
	// datastore returns them with byVersion=true).
	versions := []fleet.FleetMaintainedVersion{
		{ID: 12, Version: "149.0.2"},
		{ID: 11, Version: "147.0.9"},
		{ID: 9, Version: "148.0.1"},
		{ID: 8, Version: "147.0.5"},
	}

	teamID := uint(1)
	candidate := func(activeID uint, activeVer string) fleet.FMAAutoUpdateCandidate {
		return fleet.FMAAutoUpdateCandidate{
			TeamID:      &teamID,
			TitleID:     1,
			InstallerID: activeID,
			Version:     activeVer,
			Slug:        "chrome/darwin",
		}
	}

	cases := []struct {
		name string
		// pin: nil => no pin row (Latest); otherwise the stored expression.
		pin          *string
		active       fleet.FMAAutoUpdateCandidate
		cached       []fleet.FleetMaintainedVersion
		wantFlip     bool
		wantActiveID uint // installer ID bound active (when wantFlip)
		wantOldID    uint // installer ID whose side effects are processed
	}{
		{
			name:         "Latest advances to newest cached",
			pin:          nil,
			active:       candidate(9, "148.0.1"),
			cached:       versions,
			wantFlip:     true,
			wantActiveID: 12,
			wantOldID:    9,
		},
		{
			name:     "Latest already on newest is a no-op",
			pin:      nil,
			active:   candidate(12, "149.0.2"),
			cached:   versions,
			wantFlip: false,
		},
		{
			name:         "caret advances within major",
			pin:          new("^147"),
			active:       candidate(8, "147.0.5"),
			cached:       versions,
			wantFlip:     true,
			wantActiveID: 11, // 147.0.9, newest within major 147
			wantOldID:    8,
		},
		{
			name:     "caret never crosses major when no in-major version is cached",
			pin:      new("^147"),
			active:   candidate(12, "149.0.2"),
			cached:   []fleet.FleetMaintainedVersion{{ID: 12, Version: "149.0.2"}},
			wantFlip: false,
		},
		{
			name:     "literal pin never advances",
			pin:      new("148.0.1"),
			active:   candidate(9, "148.0.1"),
			cached:   versions,
			wantFlip: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)

			ds.ListFleetMaintainedAppActiveInstallersFunc = func(ctx context.Context) ([]fleet.FMAAutoUpdateCandidate, error) {
				return []fleet.FMAAutoUpdateCandidate{tc.active}, nil
			}
			ds.GetPinnedVersionFunc = func(ctx context.Context, tmID *uint, titleID uint) (*string, error) {
				if tc.pin == nil {
					return nil, sql.ErrNoRows
				}
				return tc.pin, nil
			}
			ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, byVersion bool) ([]fleet.FleetMaintainedVersion, error) {
				return tc.cached, nil
			}

			var gotActiveID, gotOldID uint
			ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error {
				gotActiveID = activeInstallerID
				// The cron must never write pin state, or it could clobber a
				// concurrent admin pin change.
				require.Nil(t, payload.PinnedVersion, "cron must not write the pin row")
				return nil
			}
			ds.ProcessInstallerUpdateSideEffectsFunc = func(ctx context.Context, installerID uint, wasMetadataUpdated, wasPackageUpdated bool) error {
				gotOldID = installerID
				return nil
			}

			// nil store: promote-only mode (no upstream download), exercising
			// advancement among already-cached versions.
			err := AutoUpdateFleetMaintainedApps(context.Background(), ds, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)

			require.Equal(t, tc.wantFlip, ds.SetFleetMaintainedAppActiveInstallerFuncInvoked)
			require.Equal(t, tc.wantFlip, ds.ProcessInstallerUpdateSideEffectsFuncInvoked)
			if tc.wantFlip {
				require.Equal(t, tc.wantActiveID, gotActiveID)
				require.Equal(t, tc.wantOldID, gotOldID)
			}
			// A literal pin short-circuits before querying cached versions.
			if tc.pin != nil && *tc.pin != "" && (*tc.pin)[0] != '^' {
				require.False(t, ds.GetFleetMaintainedVersionsByTitleIDFuncInvoked)
			}
		})
	}
}

func TestAutoUpdateFleetMaintainedAppsContinuesPastError(t *testing.T) {
	ds := new(mock.Store)
	teamID := uint(1)
	ds.ListFleetMaintainedAppActiveInstallersFunc = func(ctx context.Context) ([]fleet.FMAAutoUpdateCandidate, error) {
		return []fleet.FMAAutoUpdateCandidate{
			{TeamID: &teamID, TitleID: 1, InstallerID: 9, Slug: "bad/darwin"},
			{TeamID: &teamID, TitleID: 2, InstallerID: 20, Slug: "good/darwin"},
		}, nil
	}
	ds.GetPinnedVersionFunc = func(ctx context.Context, tmID *uint, titleID uint) (*string, error) {
		if titleID == 1 {
			return nil, errors.New("boom")
		}
		return nil, sql.ErrNoRows
	}
	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, tmID *uint, titleID uint, byVersion bool) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 21, Version: "2.0.0"}}, nil
	}
	var flippedTitle uint
	ds.SetFleetMaintainedAppActiveInstallerFunc = func(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, activeInstallerID uint) error {
		flippedTitle = payload.TitleID
		return nil
	}
	ds.ProcessInstallerUpdateSideEffectsFunc = func(ctx context.Context, installerID uint, wasMetadataUpdated, wasPackageUpdated bool) error {
		return nil
	}

	// The first candidate errors; the run must still process the second.
	err := AutoUpdateFleetMaintainedApps(context.Background(), ds, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	require.True(t, ds.SetFleetMaintainedAppActiveInstallerFuncInvoked)
	require.Equal(t, uint(2), flippedTitle)
}
