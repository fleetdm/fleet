package packaging

import (
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
		assert.Equal(t, tc.noFleet, outputFilename(tc.platform, tc.arch, tc.ext, Options{}))
		assert.Equal(t, tc.noFleet, outputFilename(tc.platform, tc.arch, tc.ext, Options{FleetName: "🚀 🚀"}))
	}

	t.Run("long names are cut at a character boundary", func(t *testing.T) {
		for _, opt := range []Options{
			{FleetName: strings.Repeat("a", 255)},
			{FleetName: strings.Repeat("東", 255)},
			{FleetName: strings.Repeat("ab-", 100)},
			{FleetName: strings.Repeat("東", 255), FleetID: 4294967295, OtherFleetNames: []string{strings.Repeat("東", 254)}},
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
		others []string
		want   string
	}{
		{"no other fleets", "Workstations", 12, nil, "workstations"},
		{"other fleets with different names", "Workstations", 12, []string{"Servers", "Workstations 2"}, "workstations"},
		{"other fleet differs only in case and emojis", "Workstations", 12, []string{"💻 WORKSTATIONS"}, "workstations-12"},
		{"other fleet differs only in punctuation", "Workstations (Canary)", 7, []string{"workstations-canary"}, "workstations-canary-7"},
		{"names that match only after the cut", long + "x", 3, []string{long + "y"}, strings.Repeat("a", 28) + "-3"},
		{"no ID to add", "Workstations", 0, []string{"workstations"}, "workstations"},
		{"emoji-only names stay without a fleet name", "🚀", 5, []string{"🛸"}, ""},
		// U+043E is a Cyrillic "о": it looks like the Latin "o" but is a different letter, so no collision.
		{"lookalike Cyrillic letter", "Wоrkstations", 9, []string{"Workstations"}, "wоrkstations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, fleetFilenamePart(tc.fleet, tc.id, tc.others, 30))
		})
	}
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
		{"emojis between words", "Mac💻Laptops", "mac-laptops"},
		{"emoji with variation selector and joiner", "❤️ Team 👩‍💻", "team"},
		{"keycap digit", "Floor 1️⃣", "floor-1"},
		{"only emojis", "🚀🚀", ""},
		{"only symbols", "$$$ ---", ""},
		{"empty", "", ""},
		{"accents", "Équipe Été", "équipe-été"},
		{"decomposed accents are composed", "Équipe", "équipe"},
		{"CJK", "東京 チーム", "東京-チーム"},
		{"combining marks in Devanagari", "हिन्दी टीम", "हिन्दी-टीम"},
		{"combining mark after a separator", "a ́b", "a-b"},
		{"only combining marks", "́̂", ""},
		{"zero-width non-joiner inside a word", "می‌خواهم", "میخواهم"},
		{"soft hyphen", "Work­stations", "workstations"},
		{"right-to-left override", "abc‮gpj.exe", "abcgpj-exe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, fleetNameForFilename(tc.in))
		})
	}
}
