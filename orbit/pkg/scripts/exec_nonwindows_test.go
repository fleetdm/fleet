//go:build !windows

package scripts

import (
	"context"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/stretchr/testify/require"
)

func TestExecCmdNonWindows(t *testing.T) {
	zshPath := "/bin/zsh"
	bashPath := "/bin/bash"
	if runtime.GOOS == "linux" {
		zshPath = "/usr/bin/zsh"
		bashPath = "/usr/bin/bash"
	}

	tests := []struct {
		name     string
		contents string
		output   string
		exitCode int
		error    error
	}{
		{
			name:     "no shebang",
			contents: "[ -z \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "sh shebang",
			contents: "#!/bin/sh\n[ -z \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "bash shebang",
			contents: "#!" + bashPath + "\n[ -n \"$BASH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "zsh shebang",
			contents: "#!" + zshPath + "\n[ -n \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "zsh shebang with args",
			contents: "#!" + zshPath + " -e\n[ -n \"$ZSH_VERSION\" ] && echo 1",
			output:   "1",
		},
		{
			name:     "unsupported shebang",
			contents: "#!/bin/ksh\necho 1",
			error:    fleet.ErrUnsupportedInterpreter,
			exitCode: -1,
		},
	}

	tmpDir := t.TempDir()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if strings.HasPrefix(tc.contents, "#!"+zshPath) {
				// skip if zsh is not installed
				if _, err := exec.LookPath(zshPath); err != nil {
					t.Skipf("zsh not installed: %s", err)
				}
			}
			scriptPath := strings.ReplaceAll(tc.name, " ", "_") + ".sh"
			scriptPath = filepath.Join(tmpDir, scriptPath)
			err := os.WriteFile(scriptPath, []byte(tc.contents), os.ModePerm) //nolint:gosec // ignore non-standard permissions
			require.NoError(t, err)

			output, exitCode, err := ExecCmd(context.Background(), scriptPath, nil)
			require.Equal(t, tc.output, strings.TrimSpace(string(output)))
			require.Equal(t, tc.exitCode, exitCode)
			require.ErrorIs(t, err, tc.error)
		})
	}
}

func writeTestScript(content string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "testscript*.sh")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	err = os.Chmod(tmpfile.Name(), 0o700) // nolint:gosec // G302
	if err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

func TestExecCmdTimeout(t *testing.T) {
	scriptContent := `#!/bin/sh
	sleep 5
	echo "Finished"`
	scriptPath, err := writeTestScript(scriptContent)
	if err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	defer os.Remove(scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	output, exitCode, err := ExecCmd(ctx, scriptPath, nil)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "signal: killed")
	if exitCode != -1 {
		t.Fatalf("Expected exit code -1, got: %d", exitCode)
	}
	if len(output) != 0 {
		t.Fatalf("Expected no output, got: %s", output)
	}
	require.True(t, time.Since(start) <= 5*time.Second)
}

func TestExecCmdSuccess(t *testing.T) {
	scriptContent := `#!/bin/sh
	echo "Hello, World!"`
	scriptPath, err := writeTestScript(scriptContent)
	if err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}
	defer os.Remove(scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, exitCode, err := ExecCmd(ctx, scriptPath, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Expected exit code 0, got: %d", exitCode)
	}
	expectedOutput := "Hello, World!\n"
	if string(output) != expectedOutput {
		t.Fatalf("Expected output %q, got: %q", expectedOutput, output)
	}
}

// Values are defined ahead of the body rather than substituted into it, so a
// value carrying interpreter metacharacters is data by the time it runs.
func TestExecCmdDoesNotExecuteFleetVariableValues(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "MARKER")

	for _, tc := range []struct {
		name     string
		value    string
		contents string
		// macOS splits shebang arguments and Linux doesn't, so the shebang case
		// only gets the safety assertion
		safetyOnly bool
	}{
		{"backtick", "Eng`touch " + marker + "`", "#!/bin/sh\nprintf %s \"$FLEET_VAR_HOST_UUID\"\n", false},
		{"cmd-subst", "Eng$(touch " + marker + ")", "#!/bin/sh\nprintf %s \"$FLEET_VAR_HOST_UUID\"\n", false},
		{"embedded quote", "Eng'; touch " + marker + "; echo '", "#!/bin/sh\nprintf %s \"$FLEET_VAR_HOST_UUID\"\n", false},
		{"no shebang", "Eng`touch " + marker + "`", "printf %s \"$FLEET_VAR_HOST_UUID\"\n", false},
		{"bash shebang", "Eng`touch " + marker + "`", "#!/bin/bash\nprintf %s \"$FLEET_VAR_HOST_UUID\"\n", false},
		// the kernel splits shebang arguments, so a substituted value would run
		{"shebang reference", "x`touch " + marker + "`", "#!/bin/sh -c $FLEET_VAR_HOST_UUID\nprintf %s \"$FLEET_VAR_HOST_UUID\"\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			preamble := variables.Preamble(map[string]string{"HOST_UUID": tc.value}, variables.DialectPOSIX)
			script, err := variables.InsertPreamble(tc.contents, preamble, variables.DialectPOSIX)
			require.NoError(t, err)

			path := filepath.Join(t.TempDir(), "script")
			require.NoError(t, os.WriteFile(path, []byte(script), 0o600))

			output, _, err := ExecCmd(context.Background(), path, nil)
			require.NoFileExists(t, marker)
			if tc.safetyOnly {
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.value, string(output))
		})
	}
}

// An older agent decides how to run a script from its first line, so the
// preamble has to leave that decision unchanged.
func TestPreambleDoesNotChangeInterpreterChoice(t *testing.T) {
	pre := variables.Preamble(map[string]string{"HOST_UUID": "ABC"}, variables.DialectPOSIX)

	for _, contents := range []string{
		"echo hi\n",
		"#!/bin/sh\necho hi\n",
		"#!/bin/sh -e\necho hi\n",
		"#!/bin/bash\necho hi\n",
		"#!/bin/zsh\necho hi\n",
		"#!/usr/bin/env bash\necho hi\n",
		"#!/bin/sh\r\necho hi\r\n",
		"# not a shebang\necho hi\n",
	} {
		t.Run(contents, func(t *testing.T) {
			withPreamble, err := variables.InsertPreamble(contents, pre, variables.DialectPOSIX)
			require.NoError(t, err)

			wantDirect, wantErr := fleet.ValidateShebang(contents)
			gotDirect, gotErr := fleet.ValidateShebang(withPreamble)
			require.Equal(t, wantErr, gotErr)
			require.Equal(t, wantDirect, gotDirect)

			wantKind, _, _ := fleet.ShebangInfo(contents)
			gotKind, _, _ := fleet.ShebangInfo(withPreamble)
			require.Equal(t, wantKind, gotKind)
		})
	}
}

func TestShebangInterpreter(t *testing.T) {
	for _, tc := range []struct {
		contents, interp, arg string
	}{
		{"echo 1", "", ""},
		{"#!/bin/sh\necho 1", "/bin/sh", ""},
		{"#! /bin/zsh -e\necho 1", "/bin/zsh", "-e"},
		{"#!/usr/bin/env -S python3 -u\r\nprint(1)", "/usr/bin/env", "-S python3 -u"},
		{"#!\t/bin/bash\t-x -e\n", "/bin/bash", "-x -e"},
		{"#!", "", ""},
	} {
		interp, arg := shebangInterpreter(tc.contents)
		require.Equal(t, tc.interp, interp, tc.contents)
		require.Equal(t, tc.arg, arg, tc.contents)
	}
}

func TestExecCmdMissingInterpreterFallsBackToSystemProfile(t *testing.T) {
	binDir := t.TempDir()
	marker := filepath.Join(binDir, "NIXOS")
	require.NoError(t, os.WriteFile(marker, nil, 0o600))
	origMarker, origBinDir := nixosMarkerFile, nixosSystemBinDir
	nixosMarkerFile, nixosSystemBinDir = marker, binDir
	t.Cleanup(func() { nixosMarkerFile, nixosSystemBinDir = origMarker, origBinDir })
	fakePython := filepath.Join(binDir, "python3")
	require.NoError(t, os.WriteFile(fakePython, []byte("#!/bin/sh\necho \"fake python3 $1\"\n"), 0o700)) // nolint:gosec // G306
	// a same-named interpreter on PATH must not be picked up
	pathDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(pathDir, "python3.99"), []byte("#!/bin/sh\necho from-path\n"), 0o700)) // nolint:gosec // G306
	t.Setenv("PATH", pathDir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tc := range []struct {
		name, contents, want string
	}{
		{"no args", "#!/nonexistent/fleet-test/python3\nprint(1)\n", "fake python3 "},
		{"with args", "#!/nonexistent/fleet-test/python3 -u\nprint(1)\n", "fake python3 -u\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scriptPath, err := writeTestScript(tc.contents)
			require.NoError(t, err)
			defer os.Remove(scriptPath)

			output, exitCode, err := ExecCmd(ctx, scriptPath, nil)
			require.NoError(t, err)
			require.Equal(t, 0, exitCode)
			require.True(t, strings.HasPrefix(string(output), tc.want), string(output))

			after, err := os.ReadFile(scriptPath)
			require.NoError(t, err)
			require.Equal(t, tc.contents, string(after))
		})
	}

	t.Run("not in system profile", func(t *testing.T) {
		scriptPath, err := writeTestScript("#!/nonexistent/fleet-test/python3.99\nprint(1)\n")
		require.NoError(t, err)
		defer os.Remove(scriptPath)

		_, exitCode, err := ExecCmd(ctx, scriptPath, nil)
		require.ErrorContains(t, err, "/nonexistent/fleet-test/python3.99 not found on this host")
		require.Equal(t, -1, exitCode)
	})

	t.Run("not NixOS", func(t *testing.T) {
		nixosMarkerFile = filepath.Join(binDir, "no-such-marker")
		scriptPath, err := writeTestScript("#!/nonexistent/fleet-test/python3\nprint(1)\n")
		require.NoError(t, err)
		defer os.Remove(scriptPath)

		_, exitCode, err := ExecCmd(ctx, scriptPath, nil)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Equal(t, -1, exitCode)
	})
}
