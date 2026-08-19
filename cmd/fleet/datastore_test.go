package main

import (
	"io"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout redirects os.Stdout for the duration of fn (the migration
// print helpers write there directly) and returns what was written, so test
// output stays clean and the banner wiring can be asserted.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	fn()

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func TestEvalMigrationStatus(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       *fleet.MigrationStatus
		devMode      bool
		allowMissing bool
		wantExit     bool
		wantOut      string // substring expected on stdout; empty means no output
	}{
		{
			name:     "all completed never exits",
			status:   &fleet.MigrationStatus{StatusCode: fleet.AllMigrationsCompleted},
			wantExit: false,
			wantOut:  "",
		},
		{
			name:     "all completed ignores dev and allow-missing",
			status:   &fleet.MigrationStatus{StatusCode: fleet.AllMigrationsCompleted},
			devMode:  true,
			wantExit: false,
			wantOut:  "",
		},
		{
			name:     "unknown migrations fatal only in dev",
			status:   &fleet.MigrationStatus{StatusCode: fleet.UnknownMigrations, UnknownTable: []int64{1}},
			devMode:  true,
			wantExit: true,
			wantOut:  "unrecognized migrations",
		},
		{
			name:     "unknown migrations tolerated outside dev still warns",
			status:   &fleet.MigrationStatus{StatusCode: fleet.UnknownMigrations, UnknownTable: []int64{1}},
			devMode:  false,
			wantExit: false,
			wantOut:  "unrecognized migrations",
		},
		{
			name:         "needs v4732 fix exits unless allowed",
			status:       &fleet.MigrationStatus{StatusCode: fleet.NeedsFleetv4732Fix},
			allowMissing: false,
			wantExit:     true,
			wantOut:      "automatically perform this fix",
		},
		{
			name:         "needs v4732 fix tolerated when missing allowed still warns",
			status:       &fleet.MigrationStatus{StatusCode: fleet.NeedsFleetv4732Fix},
			allowMissing: true,
			wantExit:     false,
			wantOut:      "automatically perform this fix",
		},
		{
			name:     "unknown v4732 state exits unless allowed",
			status:   &fleet.MigrationStatus{StatusCode: fleet.UnknownFleetv4732State},
			wantExit: true,
			wantOut:  "contact Fleet support",
		},
		{
			name:         "unknown v4732 state tolerated when missing allowed still warns",
			status:       &fleet.MigrationStatus{StatusCode: fleet.UnknownFleetv4732State},
			allowMissing: true,
			wantExit:     false,
			wantOut:      "contact Fleet support",
		},
		{
			name:     "some migrations completed exits unless allowed",
			status:   &fleet.MigrationStatus{StatusCode: fleet.SomeMigrationsCompleted, MissingTable: []int64{7}},
			wantExit: true,
			wantOut:  "tables=[7]",
		},
		{
			name:         "some migrations completed tolerated when missing allowed still warns",
			status:       &fleet.MigrationStatus{StatusCode: fleet.SomeMigrationsCompleted, MissingTable: []int64{7}},
			allowMissing: true,
			wantExit:     false,
			wantOut:      "tables=[7]",
		},
		{
			name:         "no migrations always exits",
			status:       &fleet.MigrationStatus{StatusCode: fleet.NoMigrationsCompleted},
			allowMissing: true,
			devMode:      true,
			wantExit:     true,
			wantOut:      "not initialized",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got bool
			out := captureStdout(t, func() {
				got = evalMigrationStatus(tc.status, tc.devMode, tc.allowMissing)
			})
			assert.Equal(t, tc.wantExit, got)
			if tc.wantOut == "" {
				assert.Empty(t, out)
			} else {
				assert.Contains(t, out, tc.wantOut)
			}
		})
	}
}
