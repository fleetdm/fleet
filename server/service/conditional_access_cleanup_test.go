package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedCleanupScriptMatchesDocs(t *testing.T) {
	docs, err := os.ReadFile("../../docs/solutions/macos/scripts/delete-duplicate-scep-certificates.sh")
	require.NoError(t, err, "the public Okta CA cleanup script must exist; if you moved it, update the test path")
	require.Equal(t, string(docs), deleteDuplicateOktaSCEPScript,
		"embedded_scripts/delete-duplicate-scep-certificates.sh has drifted from the docs copy; keep them in sync")
}

func TestBuildOktaCACleanupScript(t *testing.T) {
	cases := []struct {
		name     string
		username string
		wantOK   bool
	}{
		{"valid simple", "alice", true},
		{"valid with dot", "alice.smith", true},
		{"valid with underscore prefix", "_servicedesk", true},
		{"valid with dash", "alice-smith", true},
		{"valid mixed", "Alice.Smith-1", true},
		{"empty", "", false},
		{"starts with dash", "-alice", false},
		{"starts with dot", ".alice", false},
		{"contains space", "alice smith", false},
		{"contains slash", "alice/smith", false},
		{"contains single quote", "ali'ce", false},
		{"contains backtick", "ali`ce", false},
		{"contains semicolon", "alice;rm -rf /", false},
		{"contains shell metachar dollar", "ali$ce", false},
		{"too long", strings.Repeat("a", 32), false},
		{"exactly 31 chars", strings.Repeat("a", 31), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := buildOktaCACleanupScript(c.username)
			require.Equal(t, c.wantOK, ok)
			if !c.wantOK {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, "set -- -y -u '"+c.username+"' 'Fleet conditional access for Okta'")
			assert.Contains(t, got, deleteDuplicateOktaSCEPScript,
				"wrapped script must contain the embedded cleanup script verbatim")
		})
	}
}

func TestShellSingleQuote(t *testing.T) {
	cases := map[string]string{
		"":         "''",
		"foo":      "'foo'",
		"foo bar":  "'foo bar'",
		"al'ce":    `'al'"'"'ce'`,
		"$danger`": "'$danger`'",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			require.Equal(t, want, shellSingleQuote(in))
		})
	}
}

func TestMaybeRunOktaCACleanupScript(t *testing.T) {
	ctx := context.Background()
	const (
		hostUUID    = "host-uuid"
		commandUUID = "cmd-uuid"
		hostID      = uint(99)
		shortName   = "alice"
	)
	probeErrBoom := errors.New("boom")

	cases := []struct {
		name           string
		target         fleet.OktaCACleanupTarget
		targetOK       bool
		targetErr      error
		wantErr        bool
		wantEnqueue    bool
		wantInvalidLog bool
	}{
		{
			name:        "okta CA profile + valid user: enqueues",
			target:      fleet.OktaCACleanupTarget{HostID: hostID, UserShortName: shortName},
			targetOK:    true,
			wantEnqueue: true,
		},
		{
			name:     "not okta CA profile: no enqueue",
			targetOK: false,
		},
		{
			name:      "lookup error: propagates",
			targetErr: probeErrBoom,
			wantErr:   true,
		},
		{
			name:           "invalid username: skipped without error",
			target:         fleet.OktaCACleanupTarget{HostID: hostID, UserShortName: "ali ce"},
			targetOK:       true,
			wantInvalidLog: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.OktaCACleanupTargetForInstallCommandFunc = func(_ context.Context, hUUID, cmdUUID string) (fleet.OktaCACleanupTarget, bool, error) {
				require.Equal(t, hostUUID, hUUID)
				require.Equal(t, commandUUID, cmdUUID)
				return c.target, c.targetOK, c.targetErr
			}
			ds.NewInternalHostScriptExecutionRequestFunc = func(_ context.Context, req *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
				require.Equal(t, hostID, req.HostID)
				require.Contains(t, req.ScriptContents, "set -- -y -u '"+shortName+"' 'Fleet conditional access for Okta'")
				require.Contains(t, req.ScriptContents, deleteDuplicateOktaSCEPScript)
				require.False(t, req.SyncRequest)
				require.Nil(t, req.UserID)
				return &fleet.HostScriptResult{}, nil
			}

			svc := &MDMAppleCheckinAndCommandService{
				ds:     ds,
				logger: slog.New(slog.DiscardHandler),
			}
			err := svc.maybeRunOktaCACleanupScript(ctx, hostUUID, commandUUID)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.wantEnqueue, ds.NewInternalHostScriptExecutionRequestFuncInvoked)
		})
	}
}
