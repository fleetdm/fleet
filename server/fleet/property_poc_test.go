// Property-based testing POC for Fleet's server/fleet package.
// Demonstrates what rapid (pgregory.net/rapid) can find that table-driven
// tests miss. This file is not intended for merging as-is; the bug findings
// should be filed separately and fixes applied, then the relevant tests can
// move into the canonical test files.
//
// Run:   go test -run 'TestPBT_' ./server/fleet/ -args -rapid.checks=2000
//
// Findings:
//
//   1. BUG: MDMProfileSpecsMatch is not reflexive when a slice contains
//      two entries with the same Path. The function is documented as
//      checking that two slices "contain the same spec elements, regardless
//      of order," but MDMProfileSpecsMatch(x, x) returns false for
//      x = [{Path:"/a"}, {Path:"/a"}].
//
//   2. BUG: MDMNameFromServerURL is non-deterministic when a URL contains
//      multiple known MDM check substrings. "https://jumpcloud.awmdm.com"
//      returns "JumpCloud" on some calls and "VMware Workspace ONE" on
//      others, within a single process.
//
//   3. BUG: IsLooseEmail accepts strings that clearly are not emails,
//      specifically trailing whitespace and trailing "junk" text. The
//      function's doc comment says "has no spaces," but the regex allows
//      spaces in the segment after the domain's first dot.

package fleet

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// =============================================================================
// FINDING #1: MDMProfileSpecsMatch is not reflexive with duplicate paths
// =============================================================================

func profileSpecGen() *rapid.Generator[MDMProfileSpec] {
	// Narrow pools so duplicates are likely — that's where multiset semantics
	// get interesting.
	labelGen := rapid.SliceOfN(rapid.SampledFrom([]string{"x", "y", "z"}), 0, 3)
	pathGen := rapid.SampledFrom([]string{"/a", "/b"})
	return rapid.Custom(func(t *rapid.T) MDMProfileSpec {
		return MDMProfileSpec{
			Path:             pathGen.Draw(t, "Path"),
			Labels:           labelGen.Draw(t, "Labels"),
			LabelsIncludeAll: labelGen.Draw(t, "LabelsIncludeAll"),
			LabelsIncludeAny: labelGen.Draw(t, "LabelsIncludeAny"),
			LabelsExcludeAny: labelGen.Draw(t, "LabelsExcludeAny"),
		}
	})
}

// naiveMatch canonicalizes each spec into a string, then does multiset
// equality. This is what the function claims to do per its doc comment.
func naiveMatch(a, b []MDMProfileSpec) bool {
	if len(a) != len(b) {
		return false
	}
	toKey := func(s MDMProfileSpec) string {
		sortStrs := func(in []string) []string {
			out := append([]string{}, in...)
			slices.Sort(out)
			return out
		}
		sortedLabels := sortStrs(s.Labels)
		sortedAll := sortStrs(s.LabelsIncludeAll)
		sortedAny := sortStrs(s.LabelsIncludeAny)
		sortedExcl := sortStrs(s.LabelsExcludeAny)
		// Per MDMProfileSpecsMatch source: deprecated Labels only used when
		// LabelsIncludeAll is empty.
		effInclude := sortedAll
		if len(effInclude) == 0 {
			effInclude = sortedLabels
		}
		return s.Path + "|" + strings.Join(effInclude, ",") + "|" +
			strings.Join(sortedAny, ",") + "|" + strings.Join(sortedExcl, ",")
	}
	counts := make(map[string]int)
	for _, s := range a {
		counts[toKey(s)]++
	}
	for _, s := range b {
		counts[toKey(s)]--
	}
	for _, v := range counts {
		if v != 0 {
			return false
		}
	}
	return true
}

// Comparing a slice to itself should always be true.
// FAILS on [{Path:"/a"}, {Path:"/a"}].
func TestPBT_MDMProfileSpecsMatchReflexive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.SliceOfN(profileSpecGen(), 0, 5).Draw(t, "a")
		require.True(t, MDMProfileSpecsMatch(a, a),
			"Match(a, a) should be true by reflexivity; a=%+v", a)
	})
}

// Stronger: the implementation should agree with a naive multiset comparison.
// FAILS on a = [{Path:"/a"}, {Path:"/a"}]: impl=false, naive=true.
func TestPBT_MDMProfileSpecsMatchVsNaive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.SliceOfN(profileSpecGen(), 0, 5).Draw(t, "a")
		require.Equalf(t, naiveMatch(a, a), MDMProfileSpecsMatch(a, a),
			"disagreement comparing slice to itself; a=%+v", a)
	})
}

// =============================================================================
// FINDING #2: MDMNameFromServerURL is non-deterministic on ambiguous URLs
// =============================================================================

// Calling the function twice on the same input should always return the same
// value. FAILS on "https://jumpcloud.awmdm.com" (contains both "jumpcloud"
// and "awmdm", two different MDM provider matches).
func TestPBT_MDMNameFromServerURLDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		checks := []string{"kandji", "iru.com", "jamf", "jumpcloud", "airwatch", "awmdm", "microsoft", "simplemdm", "fleetdm", "mosyle", "other"}
		parts := rapid.SliceOfN(rapid.SampledFrom(checks), 1, 4).Draw(t, "parts")
		url := "https://" + strings.Join(parts, ".") + ".com"

		first := MDMNameFromServerURL(url)
		for range 50 {
			got := MDMNameFromServerURL(url)
			require.Equalf(t, first, got,
				"non-deterministic on %q", url)
		}
	})
}

// =============================================================================
// FINDING #3: IsLooseEmail accepts trailing whitespace and trailing junk
// =============================================================================

// The function's doc comment says "has no spaces, a single @ character...".
// The regex `^[^\s@]+@[^\s@\.]+\..+$` has `.+$` at the end, where `.` in Go
// regex matches any character EXCEPT newline — including spaces and tabs.
// Consequence: "a@b.c " and even "a@b.c this is not an email" both pass.
// This is used to validate the --end-user-email CLI flag for orbit and
// fleetctl (orbit/cmd/orbit/orbit.go:444, cmd/fleetctl/fleetctl/package.go:337)
// so a user can sneak whitespace-padded or multi-token values through.
//
// Property: if IsLooseEmail accepts `x`, appending whitespace or additional
// space-separated tokens should NOT still be accepted. Anything the doc says
// "has no spaces" should reject strings that contain them.
// FAILS on "a@b.c" + " " and "a@b.c" + "   extra text".
func TestPBT_IsLooseEmailNoTrailingSpaces(t *testing.T) {
	localGen := rapid.StringMatching(`[a-z]{1,6}`)
	domainGen := rapid.StringMatching(`[a-z]{1,6}`)
	tldGen := rapid.StringMatching(`[a-z]{1,4}`)
	trailingGen := rapid.SampledFrom([]string{" ", "\t", "  ", "   extra text", " extra@garbage.com"})

	rapid.Check(t, func(t *rapid.T) {
		// Build a known-valid minimal email first.
		base := localGen.Draw(t, "local") + "@" +
			domainGen.Draw(t, "domain") + "." +
			tldGen.Draw(t, "tld")
		require.Truef(t, IsLooseEmail(base), "base should be a valid email: %q", base)

		// Appending any amount of trailing whitespace or junk should make it
		// invalid, because an email "has no spaces" per the doc comment.
		trailing := trailingGen.Draw(t, "trailing")
		padded := base + trailing
		require.Falsef(t, IsLooseEmail(padded),
			"IsLooseEmail accepted padded input; base=%q, padded=%q", base, padded)
	})
}

// =============================================================================
// A few passing tests kept for contrast (all pass at 5000 checks).
// =============================================================================

func sign(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	default:
		return 0
	}
}

func versionStringGen() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.StringMatching(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`),
		rapid.StringMatching(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3} \([a-z]\)`),
		rapid.String(),
	)
}

// PASSES: CompareVersions is a valid total order.
func TestPBT_CompareVersionsAntisymmetric(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := versionStringGen().Draw(t, "a")
		b := versionStringGen().Draw(t, "b")
		require.Equalf(t, -sign(CompareVersions(b, a)), sign(CompareVersions(a, b)),
			"not antisymmetric on %q, %q", a, b)
	})
}

// PASSES: Preprocess is idempotent.
func TestPBT_PreprocessIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		x := rapid.String().Draw(t, "x")
		once := Preprocess(x)
		twice := Preprocess(once)
		require.Equalf(t, once, twice, "not idempotent on %q", x)
	})
}

// PASSES: MaybeExpand with an always-replace mapper matches os.Expand on
// well-formed input.
func TestPBT_MaybeExpandVsOsExpand(t *testing.T) {
	wordGen := rapid.StringMatching(`[A-Za-z_][A-Za-z0-9_]{0,4}`)
	fragGen := rapid.OneOf(
		rapid.StringMatching(`[a-z ]{0,4}`),
		rapid.Custom(func(t *rapid.T) string { return "$" + wordGen.Draw(t, "v") }),
		rapid.Custom(func(t *rapid.T) string { return "${" + wordGen.Draw(t, "v") + "}" }),
	)
	rapid.Check(t, func(t *rapid.T) {
		s := strings.Join(rapid.SliceOfN(fragGen, 0, 6).Draw(t, "frags"), "")
		meOut := MaybeExpand(s, func(_ string, _, _ int) (string, bool) { return "REPL", true })
		osOut := os.Expand(s, func(_ string) string { return "REPL" })
		require.Equalf(t, osOut, meOut, "MaybeExpand != os.Expand on %q", s)
	})
}
