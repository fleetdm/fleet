package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
)

func TestRunScriptCommand(t *testing.T) {
	_, ds := runServerWithMockedDS(t, &service.TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		// increase the default timeout to 90 seconds to match the production server
		HTTPServerConfig: &http.Server{WriteTimeout: 90 * time.Second}, // nolint:gosec
	})

	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
		return nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return nil, nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, hid uint) ([]*fleet.HostBattery, error) {
		return nil, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		require.IsType(t, fleet.ActivityTypeRanScript{}, activity)
		return nil
	}

	generateValidPath := func() string {
		return writeTmpScriptContents(t, "echo hello world", ".sh")
	}
	maxChars := strings.Repeat("a", 10001)

	type testCase struct {
		name           string
		scriptPath     func() string
		scriptResult   *fleet.HostScriptResult
		expectOutput   string
		expectErrMsg   string
		expectNotFound bool
		expectOffline  bool
		expectPending  bool
	}

	cases := []testCase{
		{
			name:          "host offline",
			scriptPath:    generateValidPath,
			expectErrMsg:  fleet.RunScriptHostOfflineErrMsg,
			expectOffline: true,
		},
		{
			name:           "host not found",
			scriptPath:     generateValidPath,
			expectErrMsg:   fleet.RunScriptHostNotFoundErrMsg,
			expectNotFound: true,
		},
		{
			name:         "invalid file type",
			scriptPath:   func() string { return writeTmpScriptContents(t, "echo hello world", ".txt") },
			expectErrMsg: fleet.RunScriptInvalidTypeErrMsg,
		},
		{
			name:         "invalid hashbang",
			scriptPath:   func() string { return writeTmpScriptContents(t, "#! /foo/bar", ".sh") },
			expectErrMsg: `Interpreter not supported. Bash scripts must run in "#!/bin/sh”.`,
		},
		{
			name: "script too long",
			scriptPath: func() string {
				return writeTmpScriptContents(t, maxChars, ".sh")
			},
			expectErrMsg: `Script is too large. It’s limited to 10,000 characters (approximately 125 lines).`,
		},
		{
			name:         "script empty",
			scriptPath:   func() string { return writeTmpScriptContents(t, "", ".sh") },
			expectErrMsg: `Script contents must not be empty.`,
		},
		{
			name:         "invalid utf8",
			scriptPath:   func() string { return writeTmpScriptContents(t, "\xff\xfa", ".sh") },
			expectErrMsg: `Wrong data format. Only plain text allowed.`,
		},
		{
			name:          "script already running",
			scriptPath:    generateValidPath,
			expectErrMsg:  fleet.RunScriptAlreadyRunningErrMsg,
			expectPending: true,
		},
		{
			name:       "script successful",
			scriptPath: generateValidPath,
			scriptResult: &fleet.HostScriptResult{
				ExitCode: ptr.Int64(0),
				Output:   "hello world",
			},
			expectOutput: `
Exit code: 0 (Script ran successfully.)

Output:

-------------------------------------------------------------------------------------

hello world

-------------------------------------------------------------------------------------
`,
		},
		{
			name:       "script failed",
			scriptPath: generateValidPath,
			scriptResult: &fleet.HostScriptResult{
				ExitCode: ptr.Int64(1),
				Output:   "",
			},
			expectOutput: `
Exit code: 1 (Script failed.)

Output:

-------------------------------------------------------------------------------------



-------------------------------------------------------------------------------------
`,
		},
		{
			name:       "script killed",
			scriptPath: generateValidPath,
			scriptResult: &fleet.HostScriptResult{
				ExitCode: ptr.Int64(-1),
				Output:   "Oh no!",
				Message:  "Timeout. Fleet stopped the script after 30 seconds to protect host performance.",
			},
			expectOutput: `
Error: Timeout. Fleet stopped the script after 30 seconds to protect host performance.

Output before timeout:

-------------------------------------------------------------------------------------

Oh no!

-------------------------------------------------------------------------------------
`,
		},
		{
			name:       "scripts disabled",
			scriptPath: generateValidPath,
			scriptResult: &fleet.HostScriptResult{
				ExitCode: ptr.Int64(-2),
				Output:   "",
				Message:  "Scripts are disabled for this host. To run scripts, deploy a Fleet installer with scripts enabled.",
			},
			expectOutput: `
Error: Scripts are disabled for this host. To run scripts, deploy a Fleet installer with scripts enabled.

`,
		},
		{
			name:       "output truncated",
			scriptPath: generateValidPath,
			scriptResult: &fleet.HostScriptResult{
				ExitCode: ptr.Int64(0),
				Output:   maxChars,
			},
			expectOutput: fmt.Sprintf(`
Exit code: 0 (Script ran successfully.)

Output:

-------------------------------------------------------------------------------------

Fleet records the last 10,000 characters to prevent downtime.

%s

-------------------------------------------------------------------------------------
`, maxChars),
		},
		{
			name:         "host timeout",
			scriptPath:   generateValidPath,
			expectErrMsg: fleet.RunScriptHostTimeoutErrMsg,
		},
	}

	setupDS := func(t *testing.T, c testCase) {
		ds.HostByIdentifierFunc = func(ctx context.Context, ident string) (*fleet.Host, error) {
			if ident != "host1" || c.expectNotFound {
				return nil, &notFoundError{}
			}
			return &fleet.Host{ID: 42, SeenTime: time.Now()}, nil
		}
		ds.HostFunc = func(ctx context.Context, hid uint) (*fleet.Host, error) {
			if hid != 42 || c.expectNotFound {
				return nil, &notFoundError{}
			}
			h := fleet.Host{ID: hid, SeenTime: time.Now()}
			if c.expectOffline {
				h.SeenTime = time.Now().Add(-time.Hour)
			}
			return &h, nil
		}
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hid uint, maxAge time.Duration) ([]*fleet.HostScriptResult, error) {
			require.Equal(t, uint(42), hid)
			if c.expectPending {
				return []*fleet.HostScriptResult{{HostID: uint(42)}}, nil
			}
			return nil, nil
		}
		ds.GetHostScriptExecutionResultFunc = func(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
			if c.scriptResult != nil {
				return c.scriptResult, nil
			}
			return &fleet.HostScriptResult{}, nil
		}
		ds.NewHostScriptExecutionRequestFunc = func(ctx context.Context, req *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
			require.Equal(t, uint(42), req.HostID)
			return &fleet.HostScriptResult{
				Hostname:       "host1",
				HostID:         req.HostID,
				ScriptContents: req.ScriptContents,
			}, nil
		}
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupDS(t, c)
			scriptPath := c.scriptPath()
			defer os.Remove(scriptPath)

			b, err := runAppNoChecks([]string{
				"run-script", "--host", "host1", "--script-path", scriptPath,
			})
			if c.expectErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.expectErrMsg)
			} else {
				require.NoError(t, err)
			}
			if c.scriptResult != nil {
				out := b.String()
				require.NoError(t, err)
				require.NotEmpty(t, out)
				require.Equal(t, c.expectOutput, out)
			} else {
				require.Empty(t, b.String())
			}
		})
	}
}

func writeTmpScriptContents(t *testing.T, scriptContents string, extension string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*"+extension)
	require.NoError(t, err)
	_, err = tmpFile.WriteString(scriptContents)
	require.NoError(t, err)
	return tmpFile.Name()
}
