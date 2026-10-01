package packaging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripRPMRelease(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		arch string
		want string
	}{
		{
			name: "amd64 conventional name",
			in:   "fleet-osquery-1.57.0-1.x86_64.rpm",
			arch: "x86_64",
			want: "fleet-osquery-1.57.0.x86_64.rpm",
		},
		{
			name: "arm64 conventional name",
			in:   "fleet-osquery-1.57.0-1.aarch64.rpm",
			arch: "aarch64",
			want: "fleet-osquery-1.57.0.aarch64.rpm",
		},
		{
			name: "version with build metadata (dots preserved, only release stripped)",
			in:   "fleet-osquery-1.57.0.20260708-1.x86_64.rpm",
			arch: "x86_64",
			want: "fleet-osquery-1.57.0.20260708.x86_64.rpm",
		},
		{
			name: "single-component name still strips release",
			in:   "orbit-1.0.0-1.x86_64.rpm",
			arch: "x86_64",
			want: "orbit-1.0.0.x86_64.rpm",
		},
		// Edge cases: inputs that don't carry the expected "-1.<arch>.rpm"
		// suffix are returned unchanged rather than mangled. In particular, a
		// version whose last component happens to look like a release
		// ("orbit-1.0.0") must not be truncated.
		{
			name: "no release segment with version is unchanged",
			in:   "orbit-1.0.0.x86_64.rpm",
			arch: "x86_64",
			want: "orbit-1.0.0.x86_64.rpm",
		},
		{
			name: "no release segment (no dash before arch) is unchanged",
			in:   "foobar.x86_64.rpm",
			arch: "x86_64",
			want: "foobar.x86_64.rpm",
		},
		{
			name: "no arch segment (no dot before ext) is unchanged",
			in:   "foobar.rpm",
			arch: "x86_64",
			want: "foobar.rpm",
		},
		{
			name: "arch mismatch is unchanged",
			in:   "fleet-osquery-1.57.0-1.aarch64.rpm",
			arch: "x86_64",
			want: "fleet-osquery-1.57.0-1.aarch64.rpm",
		},
		{
			name: "empty arch is a no-op",
			in:   "fleet-osquery-1.57.0-1.x86_64.rpm",
			arch: "",
			want: "fleet-osquery-1.57.0-1.x86_64.rpm",
		},
		{
			name: "empty filename is unchanged",
			in:   "",
			arch: "x86_64",
			want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, stripRPMRelease(tc.in, tc.arch))
		})
	}
}

func TestWriteSystemdUnit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		quota uint
		want  string
	}{
		{name: "default", quota: 0, want: "CPUQuota=20%"},
		{name: "custom", quota: 1, want: "CPUQuota=1%"},
		{name: "above one core", quota: 150, want: "CPUQuota=150%"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, writeSystemdUnit(Options{CPUQuota: tc.quota}, root))
			b, err := os.ReadFile(filepath.Join(root, "usr", "lib", "systemd", "system", "orbit.service"))
			require.NoError(t, err)
			require.Contains(t, string(b), "\n"+tc.want+"\n")
			require.Contains(t, string(b), "ExecStart=/opt/orbit/bin/orbit/orbit")
		})
	}
}
