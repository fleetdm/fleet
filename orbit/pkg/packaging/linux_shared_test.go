package packaging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

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
