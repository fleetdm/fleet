//go:build darwin

package main

import (
	"context"
	"log/slog"
	"os"
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/sigverify"
	"github.com/stretchr/testify/require"
)

func TestEvaluateDarwinSignature(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	signedNotarized := &sigverify.DarwinResult{
		Verified:            true,
		TeamID:              "M683GB7CPW",
		Identity:            "Developer ID Installer: Box, Inc. (M683GB7CPW)",
		NotarizationChecked: true,
		Notarized:           true,
		NotarizationDetail:  "accepted; source=Notarized Developer ID",
	}
	unsigned := &sigverify.DarwinResult{NoSignature: true, Detail: "no signature"}

	testCases := []struct {
		name    string
		res     *sigverify.DarwinResult
		pin     *maintained_apps.FMASignature
		wantErr string
	}{
		{
			name: "matching pin with notarization",
			res:  signedNotarized,
			pin:  &maintained_apps.FMASignature{AppleTeamID: "M683GB7CPW", Notarized: true},
		},
		{
			name: "signed, no pin recorded yet",
			res:  signedNotarized,
		},
		{
			name:    "team ID mismatch",
			res:     signedNotarized,
			pin:     &maintained_apps.FMASignature{AppleTeamID: "OTHERTEAM0"},
			wantErr: "signer identity changed",
		},
		{
			name:    "unsigned with identity pin",
			res:     unsigned,
			pin:     &maintained_apps.FMASignature{AppleTeamID: "M683GB7CPW"},
			wantErr: "installer is unsigned but the pin expects team ID",
		},
		{
			name:    "unsigned without pin",
			res:     unsigned,
			wantErr: `no "unsigned" signature pin`,
		},
		{
			name: "unsigned with unsigned pin",
			res:  unsigned,
			pin:  &maintained_apps.FMASignature{Unsigned: true, Justification: "vendor ships unsigned"},
		},
		{
			name: "unsigned pin but now validly signed",
			res:  signedNotarized,
			pin:  &maintained_apps.FMASignature{Unsigned: true, Justification: "vendor ships unsigned"},
			// warn-only: a vendor legitimately starting to sign should prompt
			// a pin update, not fail validation.
		},
		{
			name:    "unsigned pin but now carries an invalid signature",
			res:     &sigverify.DarwinResult{Verified: false, Detail: "invalid resource envelope"},
			pin:     &maintained_apps.FMASignature{Unsigned: true, Justification: "vendor ships unsigned"},
			wantErr: "invalid signature",
		},
		{
			name: "not notarized but pin requires it",
			res: &sigverify.DarwinResult{
				Verified:            true,
				TeamID:              "M683GB7CPW",
				Identity:            "Developer ID Installer: Box, Inc. (M683GB7CPW)",
				NotarizationChecked: true,
				Notarized:           false,
				NotarizationDetail:  "rejected; source=Unnotarized Developer ID",
			},
			pin:     &maintained_apps.FMASignature{AppleTeamID: "M683GB7CPW", Notarized: true},
			wantErr: "pin expects a notarization ticket",
		},
		{
			name:    "bad signature",
			res:     &sigverify.DarwinResult{Verified: false, Detail: "invalid resource envelope"},
			pin:     &maintained_apps.FMASignature{AppleTeamID: "M683GB7CPW"},
			wantErr: "signature verification failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := evaluateDarwinSignature(ctx, logger, tc.res, tc.pin)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
