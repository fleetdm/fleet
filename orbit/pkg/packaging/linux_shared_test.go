package packaging

import (
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
