package packaging

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputFilename(t *testing.T) {
	for _, tc := range []struct {
		platform, arch, ext string
		withFleet, noFleet  string
	}{
		{"macos", "", "pkg", "fleet-macos-workstations.pkg", "fleet-macos.pkg"},
		{"windows", "amd64", "msi", "fleet-windows-amd64-workstations.msi", "fleet-windows-amd64.msi"},
		{"windows", "arm64", "msi", "fleet-windows-arm64-workstations.msi", "fleet-windows-arm64.msi"},
		{"linux", "amd64", "deb", "fleet-linux-amd64-workstations.deb", "fleet-linux-amd64.deb"},
		{"linux", "arm64", "deb", "fleet-linux-arm64-workstations.deb", "fleet-linux-arm64.deb"},
		{"linux", "amd64", "rpm", "fleet-linux-amd64-workstations.rpm", "fleet-linux-amd64.rpm"},
		{"linux", "arm64", "rpm", "fleet-linux-arm64-workstations.rpm", "fleet-linux-arm64.rpm"},
		{"linux", "amd64", "pkg.tar.zst", "fleet-linux-amd64-workstations.pkg.tar.zst", "fleet-linux-amd64.pkg.tar.zst"},
		{"linux", "arm64", "pkg.tar.zst", "fleet-linux-arm64-workstations.pkg.tar.zst", "fleet-linux-arm64.pkg.tar.zst"},
	} {
		assert.Equal(t, tc.withFleet, outputFilename(tc.platform, tc.arch, tc.ext, Options{FleetName: "Workstations"}))
		// A name that cleans up to nothing never leaves a trailing dash (fleet-windows-amd64-.msi), even when another
		// fleet's name is just as blank.
		for _, blank := range []string{"", "\U0001F680 \U0001F680", "$$$ ---", "\u202e\u200d"} {
			assert.Equal(t, tc.noFleet, outputFilename(tc.platform, tc.arch, tc.ext, Options{FleetName: blank}), blank)
			assert.Equal(t, tc.noFleet, outputFilename(tc.platform, tc.arch, tc.ext,
				Options{FleetName: blank, FleetID: 7, OtherFleets: []Fleet{{ID: 8, Name: "\U0001F6F8"}}}), blank)
		}
	}

	t.Run("long names are cut at a character boundary", func(t *testing.T) {
		for _, opt := range []Options{
			{FleetName: strings.Repeat("a", 255)},
			{FleetName: strings.Repeat("東", 255)},
			{FleetName: strings.Repeat("ab-", 100)},
			{FleetName: strings.Repeat("東", 255), FleetID: 4294967295, OtherFleets: []Fleet{{ID: 1, Name: strings.Repeat("東", 254)}}},
		} {
			for _, ext := range []string{"pkg.tar.zst", "msi"} {
				got := outputFilename("linux", "arm64", ext, opt)
				assert.LessOrEqual(t, len(got), maxFilenameBytes, got)
				assert.True(t, utf8.ValidString(got), got)
				assert.True(t, strings.HasPrefix(got, "fleet-linux-arm64-"), got)
				assert.True(t, strings.HasSuffix(got, "."+ext), got)
				assert.NotContains(t, got, "-.", got)
			}
		}
		assert.Len(t, outputFilename("windows", "amd64", "msi", Options{FleetName: strings.Repeat("a", 255)}), maxFilenameBytes)
	})
}

func TestFleetFilenamePart(t *testing.T) {
	long := strings.Repeat("a", 80)
	for _, tc := range []struct {
		name   string
		fleet  string
		id     uint
		others []Fleet
		want   string
	}{
		{"no other fleets", "Workstations", 12, nil, "workstations"},
		{"other fleets with different names", "Workstations", 12, []Fleet{{1, "Servers"}, {2, "Workstations 2"}}, "workstations"},
		{"other fleet differs only in case and emojis", "Workstations", 12, []Fleet{{1, "\U0001F4BB WORKSTATIONS"}}, "workstations-12"},
		{"other fleet differs only in punctuation", "Workstations (Canary)", 7, []Fleet{{1, "workstations-canary"}}, "workstations-canary-7"},
		{"other fleet differs only by a long s", "Workstations", 4, []Fleet{{1, "Work\u017ftations"}}, "workstations-4"},
		{"invisible joiner doesn't dodge the ID", "Workstations\u034f!", 6, []Fleet{{1, "Workstations"}}, "workstations-6"},
		{"names that match only after the cut", long + "x", 3, []Fleet{{1, long + "y"}}, strings.Repeat("a", 28) + "-3"},
		{"name equal to another fleet's name with its ID", "foo-1", 3, []Fleet{{1, "foo"}, {2, "FOO"}}, "foo-1-3"},
		{"name with a number next to a fleet that keeps its name", "Workstations 2", 5, []Fleet{{1, "Workstations"}}, "workstations-2"},
		{"no ID to add", "Workstations", 0, []Fleet{{1, "workstations"}}, "workstations"},
		{"emoji-only names stay without a fleet name", "\U0001F680", 5, []Fleet{{1, "\U0001F6F8"}}, ""},
		// U+043E is a Cyrillic "о": it looks like the Latin "o" but is a different letter, so no collision.
		{"lookalike Cyrillic letter", "W\u043erkstations", 9, []Fleet{{1, "Workstations"}}, "w\u043erkstations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, fleetFilenamePart(Fleet{ID: tc.id, Name: tc.fleet}, tc.others, 30))
		})
	}

	t.Run("every fleet gets its own part, whatever the order", func(t *testing.T) {
		fleets := []Fleet{{1, "foo"}, {2, "FOO"}, {3, "foo-1"}, {9, "foo-1-3"}, {4, "Workstations"}, {5, "Workstations 2"}}
		want := map[uint]string{1: "foo-1", 2: "foo-2", 3: "foo-1-3", 9: "foo-1-3-9", 4: "workstations", 5: "workstations-2"}
		for i, f := range fleets {
			others := append(slices.Clone(fleets[:i]), fleets[i+1:]...)
			require.Equal(t, want[f.ID], fleetFilenamePart(f, others, 30), f.Name)
			slices.Reverse(others)
			require.Equal(t, want[f.ID], fleetFilenamePart(f, others, 30), f.Name)
		}
	})

	t.Run("no room left after the cut", func(t *testing.T) {
		require.Empty(t, fleetFilenamePart(Fleet{Name: "Workstations"}, nil, 0))
		// Too little room for the name next to the ID: only the ID, with no dash in front.
		require.Equal(t, "12", fleetFilenamePart(Fleet{ID: 12, Name: "Workstations"}, []Fleet{{1, "workstations"}}, 3))
	})
}

func TestFleetNameForFilename(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"simple", "Workstations", "workstations"},
		{"spaces and punctuation collapse", "  Workstations -- (Canary)  ", "workstations-canary"},
		{"path separators and dots", `../../tmp/x\y`, "tmp-x-y"},
		{"windows-reserved characters", `a:b*c?d"e<f>g|h`, "a-b-c-d-e-f-g-h"},
		{"control characters", "a\tb\nc\x00d", "a-b-c-d"},
		{"emojis between words", "Mac\U0001F4BBLaptops", "mac-laptops"},
		{"emoji with variation selector and joiner", "\u2764\ufe0f Team \U0001F469\u200d\U0001F4BB", "team"},
		{"keycap digit", "Floor 1\ufe0f\u20e3", "floor-1"},
		{"only emojis", "\U0001F680\U0001F680", ""},
		{"only symbols", "$$$ ---", ""},
		{"empty", "", ""},
		{"accents", "Équipe Été", "équipe-été"},
		{"decomposed accents are composed", "Équipe", "équipe"},
		{"CJK", "東京 チーム", "東京-チーム"},
		{"combining marks in Devanagari", "हिन्दी टीम", "हिन्दी-टीम"},
		{"combining mark after a separator", "a \u0301b", "a-b"},
		{"only combining marks", "́̂", ""},
		{"zero-width non-joiner inside a word", "می\u200cخواهم", "میخواهم"},
		{"soft hyphen", "Work\u00adstations", "workstations"},
		{"right-to-left override", "abc\u202egpj.exe", "abcgpj-exe"},
		{"invisible letters and marks", "Work\u034fstations\u3164\u17b4", "workstations"},
		{"accent split from its letter by a zero-width character", "Cafe\u200c\u0301", "caf\u00e9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, fleetNameForFilename(tc.in))
		})
	}
}
