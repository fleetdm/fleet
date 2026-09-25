package buildinfo

import (
	"strings"
	"testing"
)

// An unstamped build must be obviously unstamped rather than looking like a
// release someone can go and find.
func TestUnstampedDefaults(t *testing.T) {
	got := Current()
	if got.Version != "dev" {
		t.Errorf("Version = %q, want %q", got.Version, "dev")
	}
	if got.Commit != "unknown" || got.Date != "unknown" {
		t.Errorf("Commit/Date = %q/%q, want unknown", got.Commit, got.Date)
	}
}

func TestSummary(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Info
		want string
	}{
		{
			name: "stamped",
			in:   Info{Version: "1.1.0", Commit: "a1b2c3d", Date: "2026-08-14T09:12:03Z"},
			want: "1.1.0 (a1b2c3d, built 2026-08-14T09:12:03Z)",
		},
		{
			name: "dirty tree",
			in:   Info{Version: "1.1.0", Commit: "a1b2c3d-dirty", Date: "2026-08-14T09:12:03Z"},
			want: "1.1.0 (a1b2c3d-dirty, built 2026-08-14T09:12:03Z)",
		},
		{
			name: "unstamped keeps the date out of the way",
			in:   Info{Version: "dev", Commit: "unknown", Date: "unknown"},
			want: "dev (unknown)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Summary(); got != tc.want {
				t.Errorf("Summary() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The ldflags in build/darwin/Taskfile.yml address these by name; renaming one
// silently produces an unstamped build, since -X on a missing symbol is not an
// error.
func TestStampedVariableNames(t *testing.T) {
	for name, v := range map[string]string{"version": version, "commit": commit, "date": date} {
		if strings.TrimSpace(v) == "" {
			t.Errorf("%s is empty; -X targets must stay non-empty so an unstamped build is visible", name)
		}
	}
}
