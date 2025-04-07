//go:build darwin
// +build darwin

package tcc_access

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

// TestGenerate tests the tcc_access table generation.
func TestGenerate(t *testing.T) {
	tccPathPrefix = "./testdata"
	tccPathSuffix = "/test-TCC.db"

	overrideCommand(t, "dscl", "testUser1 1 \ntestUser2 2\n")

	rows, err := Generate(context.Background(), table.QueryContext{})
	require.NoError(t, err)

	require.Len(t, rows, 93)

	// Check "uid" of the returned rows match the entries in the TCC files.
	for _, row := range rows {
		switch {
		case strings.HasPrefix(row["service"], "test-sys-service-"):
			require.Equal(t, "0", row["uid"])
		case strings.HasPrefix(row["service"], "test-u1-service-"):
			require.Equal(t, "1", row["uid"])
		case strings.HasPrefix(row["service"], "test-u2-service-"):
			require.Equal(t, "2", row["uid"])
		}
	}

	rows, err = Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			"uid": {
				Affinity: table.ColumnTypeText,
				Constraints: []table.Constraint{
					{
						Operator:   table.OperatorEquals,
						Expression: "1",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 31)
	for _, row := range rows {
		serviceName := row["service"]
		require.Contains(t, serviceName, "u1-service")
		require.NotContains(t, serviceName, "u2-service")
		require.NotContains(t, serviceName, "sys-service")
	}
}

// overrideCommand allows us to override a system command (just during the execution
// of the test) by a script that prints the given output.
func overrideCommand(t *testing.T, cmdName string, output string) {
	tmpDir := t.TempDir()
	pathValue := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+os.ExpandEnv("$PATH"))
	t.Cleanup(func() {
		os.Setenv("PATH", pathValue)
	})
	cmdPath := filepath.Join(tmpDir, cmdName)
	scriptContent := []byte(fmt.Sprintf("#!/bin/sh\nprintf '%%s' \"%s\"", output))
	err := os.WriteFile(cmdPath, scriptContent, 0o744) //nolint:gosec
	require.NoError(t, err)
}
