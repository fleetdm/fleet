//go:build darwin || linux

package sshd_config

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fake sshd builds. Each records its arguments, one invocation per line, so
// tests can assert which flags were tried.

// modernSSHD mirrors sshd 9.3+: -G works, and -C is refused without -T.
const modernSSHD = `#!/bin/sh
echo "$*" >> "$(dirname "$0")/calls"
case " $* " in
  *" -C "*)
    case " $* " in *" -T "*) ;; *)
      echo "Config test connection parameter (-C) provided without test mode (-T)" >&2
      exit 1 ;;
    esac ;;
esac
case "$1" in
  -G)
    echo "port 22"
    case " $* " in
      *" user=alice"*) echo "passwordauthentication no" ;;
      *) echo "passwordauthentication yes" ;;
    esac ;;
  *) exit 1 ;;
esac
`

// legacySSHD mirrors sshd before 9.3: -G is not an option at all.
const legacySSHD = `#!/bin/sh
echo "$*" >> "$(dirname "$0")/calls"
case "$1" in
  -G)
    echo "sshd: illegal option -- G" >&2
    echo "usage: sshd [-46DdeiqTt] [-C connection_spec]" >&2
    exit 1 ;;
  -T)
    echo "port 2222"
    echo "permitrootlogin without-password" ;;
esac
`

// brokenSSHD has an invalid configuration, which both flags report.
const brokenSSHD = `#!/bin/sh
echo "$*" >> "$(dirname "$0")/calls"
echo "/etc/ssh/sshd_config: line 12: Bad configuration option: Foo" >&2
echo "/etc/ssh/sshd_config: terminating, 1 bad configuration options" >&2
exit 255
`

func writeFakeSSHD(t *testing.T, script string) (sshd sshdBinary, calls func() []string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sshd")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755)) //nolint:gosec // test script must be executable
	return sshdBinary{path: path, id: path}, func() []string {
		data, err := os.ReadFile(filepath.Join(dir, "calls"))
		if err != nil {
			return nil
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}
}

func TestDumpConfig(t *testing.T) {
	tests := []struct {
		name      string
		script    string
		spec      string
		wantCalls []string
		wantOut   string
		wantErr   string
	}{
		{
			name:      "modern, global config",
			script:    modernSSHD,
			wantCalls: []string{"-G"},
			wantOut:   "port 22\npasswordauthentication yes\n",
		},
		{
			name:      "modern, connection spec adds -T",
			script:    modernSSHD,
			spec:      "user=alice,host=h,addr=10.0.0.1",
			wantCalls: []string{"-G -T -C user=alice,host=h,addr=10.0.0.1"},
			wantOut:   "port 22\npasswordauthentication no\n",
		},
		{
			name:      "legacy falls back to -T",
			script:    legacySSHD,
			wantCalls: []string{"-G", "-T"},
			wantOut:   "port 2222\npermitrootlogin without-password\n",
		},
		{
			name:      "legacy falls back to -T with spec",
			script:    legacySSHD,
			spec:      "user=root",
			wantCalls: []string{"-G -T -C user=root", "-T -C user=root"},
			wantOut:   "port 2222\npermitrootlogin without-password\n",
		},
		{
			name:      "broken config is reported, not retried",
			script:    brokenSSHD,
			wantCalls: []string{"-G"},
			wantErr:   "-G: /etc/ssh/sshd_config: line 12: Bad configuration option: Foo /etc/ssh/sshd_config: terminating, 1 bad configuration options",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sshd, calls := writeFakeSSHD(t, tt.script)

			out, err := dumpConfig(t.Context(), sshd, tt.spec)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.True(t, strings.HasSuffix(err.Error(), tt.wantErr), "error %q lacks suffix %q", err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantOut, out)
			assert.Equal(t, tt.wantCalls, calls())
		})
	}
}

func TestDumpConfigRemembersLegacy(t *testing.T) {
	t.Parallel()
	sshd, calls := writeFakeSSHD(t, legacySSHD)

	for range 2 {
		_, err := dumpConfig(t.Context(), sshd, "")
		require.NoError(t, err)
	}
	assert.Equal(t, []string{"-G", "-T", "-T"}, calls())

	// An upgraded binary has a new id, so -G is tried again.
	sshd.id += ":upgraded"
	_, err := dumpConfig(t.Context(), sshd, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"-G", "-T", "-T", "-G", "-T"}, calls())
}

// echoSSHD mirrors how sshd rejects a malformed connection spec: it prints the
// spec back on stderr.
const echoSSHD = `#!/bin/sh
echo "-" >> "$(dirname "$0")/calls"
case " $* " in
  *" -C "*) shift 3; echo "Invalid test mode specification $1" >&2; exit 255 ;;
esac
echo "port 22"
`

// TestDumpConfigSpecCantForceLegacy checks that a query can't make the table
// skip -G for later queries by passing a spec whose echo looks like a getopt
// error.
func TestDumpConfigSpecCantForceLegacy(t *testing.T) {
	t.Parallel()
	sshd, calls := writeFakeSSHD(t, echoSSHD)

	for _, spec := range []string{"usage: sshd", "x\nusage: sshd", "x\nsshd: illegal option -- G"} {
		_, err := dumpConfig(t.Context(), sshd, spec)
		require.ErrorContains(t, err, "Invalid test mode specification")
	}
	_, legacy := lacksG.Load(sshd.id)
	assert.False(t, legacy)

	n := len(calls())
	out, err := dumpConfig(t.Context(), sshd, "")
	require.NoError(t, err)
	assert.Equal(t, "port 22\n", out)
	assert.Len(t, calls(), n+1, "a spec-less dump should run -G once")
}

func TestRunSSHDEmptyEnvironment(t *testing.T) {
	t.Setenv("SSHD_CONFIG_TEST_LEAK", "1")
	sshd, _ := writeFakeSSHD(t, "#!/bin/sh\nenv\n")

	out, _, err := runSSHD(t.Context(), sshd.path, nil)
	require.NoError(t, err)
	assert.NotContains(t, out, "SSHD_CONFIG_TEST_LEAK")
}

// useFakeSSHD makes Generate find only the given fake sshd.
func useFakeSSHD(t *testing.T, sshd sshdBinary) {
	origPaths, origResolve := sshdPaths, resolveSSHD
	t.Cleanup(func() { sshdPaths, resolveSSHD = origPaths, origResolve })
	sshdPaths = []string{filepath.Join(filepath.Dir(sshd.path), "missing"), sshd.path}
	resolveSSHD = func(path string) (string, fs.FileInfo, error) {
		info, err := os.Lstat(path)
		return path, info, err
	}
}

func TestGenerate(t *testing.T) {
	sshd, calls := writeFakeSSHD(t, modernSSHD)
	useFakeSSHD(t, sshd)

	global := []map[string]string{
		{"keyword": "port", "value": "22", "connection_spec": ""},
		{"keyword": "passwordauthentication", "value": "yes", "connection_spec": ""},
	}
	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	assert.Equal(t, global, rows)

	// A rescan within cacheTTL reuses the dump.
	rows, err = Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	assert.Equal(t, global, rows)
	assert.Equal(t, []string{"-G"}, calls())

	rows, err = Generate(t.Context(), specQueryContext(
		table.Constraint{Operator: table.OperatorEquals, Expression: "user=alice"},
		table.Constraint{Operator: table.OperatorEquals, Expression: "user=bob"},
	))
	require.NoError(t, err)
	assert.Equal(t, []map[string]string{
		{"keyword": "port", "value": "22", "connection_spec": "user=alice"},
		{"keyword": "passwordauthentication", "value": "no", "connection_spec": "user=alice"},
		{"keyword": "port", "value": "22", "connection_spec": "user=bob"},
		{"keyword": "passwordauthentication", "value": "yes", "connection_spec": "user=bob"},
	}, rows)
	assert.Equal(t, []string{"-G", "-G -T -C user=alice", "-G -T -C user=bob"}, calls())

	// A non-executable file at a candidate path is skipped.
	require.NoError(t, os.Chmod(sshd.path, 0o600))
	rows, err = Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestGenerateErrorNotCached(t *testing.T) {
	sshd, calls := writeFakeSSHD(t, brokenSSHD)
	useFakeSSHD(t, sshd)

	for range 2 {
		_, err := Generate(t.Context(), table.QueryContext{})
		require.ErrorContains(t, err, "Bad configuration option: Foo")
	}
	assert.Equal(t, []string{"-G", "-G"}, calls())
}

func TestSSHDPaths(t *testing.T) {
	require.NotEmpty(t, sshdPaths)
	for _, p := range sshdPaths {
		assert.True(t, filepath.IsAbs(p), "sshd path %q must be absolute", p)
	}
}
