package nvd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/stretchr/testify/require"
)

func TestLoadCVEResolvedVersions(t *testing.T) {
	t.Run("missing file returns empty set", func(t *testing.T) {
		overrides, err := loadCVEResolvedVersions(filepath.Join(t.TempDir(), "does-not-exist.json"))
		require.NoError(t, err)
		require.Empty(t, overrides)
		// A nil/empty map must be safe to query.
		require.Empty(t, overrides.FindResolvedVersion("CVE-2025-63389",
			&wfn.Attributes{Vendor: "ollama", Product: "ollama", Version: "0.12.3"}))
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))
		_, err := loadCVEResolvedVersions(path)
		require.Error(t, err)
	})

	t.Run("groups entries by cve", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "overrides.json")
		require.NoError(t, os.WriteFile(path, []byte(`[
			{"cve":"CVE-1","vendor":"v1","product":"p1","resolved_in_version":"1.0"},
			{"cve":"CVE-1","vendor":"v2","product":"p2","resolved_in_version":"2.0"},
			{"cve":"CVE-2","vendor":"v3","product":"p3","resolved_in_version":"3.0"}
		]`), 0o600))
		overrides, err := loadCVEResolvedVersions(path)
		require.NoError(t, err)
		require.Len(t, overrides["CVE-1"], 2)
		require.Len(t, overrides["CVE-2"], 1)
	})

	t.Run("in-repo seed feed loads and contains the ollama override", func(t *testing.T) {
		overrides, err := loadCVEResolvedVersions("cve_resolved_versions.json")
		require.NoError(t, err)
		require.Equal(t, "0.12.4", overrides.FindResolvedVersion("CVE-2025-63389",
			&wfn.Attributes{Vendor: "ollama", Product: "ollama", Version: "0.12.3"}))
	})
}

func TestCVEResolvedVersionsFindResolvedVersion(t *testing.T) {
	overrides := CVEResolvedVersions{
		"CVE-2025-63389": {
			{CVE: "CVE-2025-63389", Vendor: "ollama", Product: "ollama", ResolvedInVersion: "0.12.4"},
		},
	}

	tests := []struct {
		name string
		cve  string
		meta *wfn.Attributes
		want string
	}{
		{
			name: "vulnerable version gets the fix version",
			cve:  "CVE-2025-63389",
			meta: &wfn.Attributes{Vendor: "ollama", Product: "ollama", Version: "0.12.3"},
			want: "0.12.4",
		},
		{
			name: "older vulnerable version still gets the fix version",
			cve:  "CVE-2025-63389",
			meta: &wfn.Attributes{Vendor: "ollama", Product: "ollama", Version: "0.12.0"},
			want: "0.12.4",
		},
		{
			name: "version at the fix returns empty",
			cve:  "CVE-2025-63389",
			meta: &wfn.Attributes{Vendor: "ollama", Product: "ollama", Version: "0.12.4"},
			want: "",
		},
		{
			name: "version above the fix returns empty",
			cve:  "CVE-2025-63389",
			meta: &wfn.Attributes{Vendor: "ollama", Product: "ollama", Version: "0.13.0"},
			want: "",
		},
		{
			name: "different product returns empty",
			cve:  "CVE-2025-63389",
			meta: &wfn.Attributes{Vendor: "notollama", Product: "notollama", Version: "0.12.0"},
			want: "",
		},
		{
			name: "unknown cve returns empty",
			cve:  "CVE-0000-00000",
			meta: &wfn.Attributes{Vendor: "ollama", Product: "ollama", Version: "0.12.0"},
			want: "",
		},
		{
			name: "nil meta returns empty",
			cve:  "CVE-2025-63389",
			meta: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, overrides.FindResolvedVersion(tt.cve, tt.meta))
		})
	}
}
