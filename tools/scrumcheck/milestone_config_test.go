package main

import (
	"reflect"
	"testing"
)

// TestParseRepoFromIssueURL verifies repository owner/name extraction from
// supported GitHub issue/PR URLs and invalid inputs.
func TestParseRepoFromIssueURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		in        string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "issues URL",
			in:        "https://github.com/fleetdm/fleet/issues/12345",
			wantOwner: "fleetdm",
			wantRepo:  "fleet",
		},
		{
			name:      "pull URL",
			in:        "https://github.com/fleetdm/fleet/pull/999",
			wantOwner: "fleetdm",
			wantRepo:  "fleet",
		},
		{
			name:      "invalid path",
			in:        "https://github.com/fleetdm/fleet/discussions/55",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "bad URL",
			in:        "://not-valid-url",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotOwner, gotRepo := parseRepoFromIssueURL(tc.in)
			if gotOwner != tc.wantOwner || gotRepo != tc.wantRepo {
				t.Fatalf("parseRepoFromIssueURL(%q) = (%q, %q), want (%q, %q)", tc.in, gotOwner, gotRepo, tc.wantOwner, tc.wantRepo)
			}
		})
	}
}

// TestIntListFlagSet verifies the custom int list flag parses comma-separated
// values and supports multiple Set calls.
func TestIntListFlagSet(t *testing.T) {
	t.Parallel()

	var f intListFlag
	if err := f.Set("71, 97"); err != nil {
		t.Fatalf("unexpected error from Set: %v", err)
	}
	if err := f.Set("105"); err != nil {
		t.Fatalf("unexpected error from Set second call: %v", err)
	}

	want := []int{71, 97, 105}
	if !reflect.DeepEqual([]int(f), want) {
		t.Fatalf("flag values = %#v, want %#v", []int(f), want)
	}
}

// TestIntListFlagSetInvalid verifies invalid numeric values are rejected.
func TestIntListFlagSetInvalid(t *testing.T) {
	t.Parallel()

	var f intListFlag
	if err := f.Set("71,nope"); err == nil {
		t.Fatal("expected error for invalid number, got nil")
	}
}

// TestTitleCaseWords verifies the helper normalizes spacing/casing to title case.
func TestTitleCaseWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{in: "ready to estimate", want: "Ready To Estimate"},
		{in: "  MIXED   cASe ", want: "Mixed Case"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := titleCaseWords(tc.in)
			if got != tc.want {
				t.Fatalf("titleCaseWords(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
