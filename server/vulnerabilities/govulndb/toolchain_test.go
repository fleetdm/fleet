package govulndb

import (
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// publishedRules compiles the rules the mirror publishes.
func publishedRules(t *testing.T) *ToolchainRules {
	t.Helper()
	rules, err := NewToolchainRules(PublishedToolchainInstalls())
	require.NoError(t, err)
	return rules
}

// chocolateyRules identify only Chocolatey's golang package, so a test can tell them apart from
// the published rules.
func chocolateyRules() *ToolchainInstalls {
	return &ToolchainInstalls{
		Version: ToolchainInstallsVersion,
		Rules: []ToolchainInstallRule{{
			Source:   "chocolatey_packages",
			Names:    []string{"golang"},
			Versions: []VersionSource{{Field: "version", Pattern: `^(.+)$`}},
		}},
	}
}

// Servers apply the rules as published, and NVD skips everything they claim, so a broad or
// misdirected rule would remove vulnerabilities from customer hosts.
func TestPublishedToolchainInstalls(t *testing.T) {
	installs := PublishedToolchainInstalls()
	require.Equal(t, ToolchainInstallsVersion, installs.Version)
	require.NotEmpty(t, installs.Rules)
	publishedRules(t)

	for _, r := range installs.Rules {
		t.Run(r.Source, func(t *testing.T) {
			require.NotEmpty(t, r.Source)
			// Distributions backport fixes to deb and rpm packages without changing the upstream
			// version, and Go binaries have their own matcher.
			require.NotContains(t, []string{"deb_packages", "rpm_packages", goBinariesSource}, r.Source)
			require.NotEmpty(t, append(r.Names, r.NamePrefixes...))
			require.NotContains(t, r.Names, "")
			for _, prefix := range r.NamePrefixes {
				require.GreaterOrEqual(t, len(prefix), 3, "name prefix %q claims unrelated software", prefix)
			}
			require.NotEmpty(t, r.Versions)
		})
	}
}

// NewToolchainRules refuses only rules it cannot apply; TestPublishedToolchainInstalls checks
// that the published rules are sensible.
func TestNewToolchainRulesRejects(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func(*ToolchainInstalls)
		wantErr string
	}{
		{
			name:    "unknown version field",
			mutate:  func(ti *ToolchainInstalls) { ti.Rules[0].Versions[0].Field = "release" },
			wantErr: "field",
		},
		{
			name:    "pattern that does not compile",
			mutate:  func(ti *ToolchainInstalls) { ti.Rules[0].Versions[0].Pattern = `^(.+$` },
			wantErr: "pattern",
		},
		{
			name:    "pattern without a capture group",
			mutate:  func(ti *ToolchainInstalls) { ti.Rules[0].Versions[0].Pattern = `^.+$` },
			wantErr: "capture group",
		},
		{
			name:    "pattern with two capture groups",
			mutate:  func(ti *ToolchainInstalls) { ti.Rules[0].Versions[0].Pattern = `^(\d+)\.(.+)$` },
			wantErr: "capture group",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installs := chocolateyRules()
			tc.mutate(installs)

			_, err := NewToolchainRules(installs)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestToolchainRulesFrom(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	homebrewGo := &fleet.Software{Name: "go", Source: "homebrew_packages", Version: "1.22.4"}
	chocolateyGo := &fleet.Software{Name: "golang", Source: "chocolatey_packages", Version: "1.22.4"}

	requireNone := func(t *testing.T, rules *ToolchainRules) {
		t.Helper()
		require.False(t, rules.IsToolchainInstall(homebrewGo))
		require.False(t, rules.IsToolchainInstall(chocolateyGo))
	}

	t.Run("no artifact", func(t *testing.T) {
		requireNone(t, ToolchainRulesFrom(t.Context(), nil, logger))
	})

	t.Run("artifact published before the rules were", func(t *testing.T) {
		requireNone(t, ToolchainRulesFrom(t.Context(), testArtifact(), logger))
	})

	t.Run("published rules", func(t *testing.T) {
		artifact := testArtifact()
		artifact.ToolchainInstalls = chocolateyRules()

		rules := ToolchainRulesFrom(t.Context(), artifact, logger)
		require.True(t, rules.IsToolchainInstall(chocolateyGo))
		require.False(t, rules.IsToolchainInstall(homebrewGo))
	})

	// A newer rule format may mean something this server cannot apply.
	t.Run("published rules in another format", func(t *testing.T) {
		artifact := testArtifact()
		artifact.ToolchainInstalls = chocolateyRules()
		artifact.ToolchainInstalls.Version = "2"

		requireNone(t, ToolchainRulesFrom(t.Context(), artifact, logger))
	})
}
