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
		assert.Equal(t, tc.withFleet, outputFilename(tc.platform, tc.arch, "Workstations", tc.ext))
		assert.Equal(t, tc.noFleet, outputFilename(tc.platform, tc.arch, "", tc.ext))
		assert.Equal(t, tc.noFleet, outputFilename(tc.platform, tc.arch, "🚀 🚀", tc.ext))
	}

	t.Run("long names are cut at a character boundary", func(t *testing.T) {
		for _, name := range []string{strings.Repeat("a", 255), strings.Repeat("東", 255), strings.Repeat("ab-", 100)} {
			got := outputFilename("linux", "arm64", name, "pkg.tar.zst")
			assert.LessOrEqual(t, len(got), maxFilenameBytes, got)
			assert.True(t, utf8.ValidString(got), got)
			assert.True(t, strings.HasPrefix(got, "fleet-linux-arm64-"), got)
			assert.True(t, strings.HasSuffix(got, ".pkg.tar.zst"), got)
			assert.NotContains(t, got, "-.", got)
		}
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
