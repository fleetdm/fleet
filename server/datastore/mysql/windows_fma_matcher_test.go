package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// Patch queries as the catalog manifests generate them.
const (
	firefoxPatchQuery    = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Mozilla Firefox (x64 en-US)' AND publisher = 'Mozilla' AND version_compare(version, '160.0.1') < 0);"
	firefoxESRPatchQuery = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox%ESR%' AND publisher = 'Mozilla' AND version_compare(version, '153.8.0') < 0);"
	dotnet8PatchQuery    = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Microsoft .NET Runtime - 8.%' AND name LIKE '%(x64)' AND publisher = 'Microsoft Corporation' AND version_compare(version, '8.0.34') < 0);"
	python313PatchQuery  = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Python 3.13.% (64-bit)' AND publisher = 'Python Software Foundation' AND version_compare(version, '3.13.17150.0') < 0);"
	python314PatchQuery  = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Python 3.14.% (64-bit)' AND publisher = 'Python Software Foundation' AND version_compare(version, '3.14.10150.0') < 0);"
	qgisPatchQuery       = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'QGIS %' AND publisher = 'QGIS.org' AND version NOT LIKE '3.44.%' AND version_compare(version, '4.2.3') < 0);"
	qgisLTRPatchQuery    = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'QGIS %' AND publisher = 'QGIS.org' AND version LIKE '3.44.%' AND version_compare(version, '3.44.15') < 0);"
	realVNCPatchQuery    = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE ((name LIKE 'RealVNC Viewer %' OR name LIKE 'RealVNC Connect Viewer %') AND publisher = 'RealVNC') AND version_compare(version, '8.6.2.18') < 0);"
	granolaPatchQuery    = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Granola %' AND publisher = 'Granola' AND version_compare(version, '7.821.0') < 0);"
	burpPatchQuery       = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE ((name LIKE 'Burp Suite 2%' OR name LIKE 'Burp Suite Professional %' OR name LIKE 'Burp Suite Community Edition %') AND publisher = 'PortSwigger Web Security') AND version_compare(version, '2026.11.2') < 0);"
	zoomPatchQuery       = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Zoom Workplace%' AND publisher = 'Zoom' AND version_compare(version, '7.2.51704') < 0);"
	zoomARMPatchQuery    = "SELECT 1 WHERE EXISTS (SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Zoom Workplace (ARM64)' AND publisher = 'Zoom' AND version_compare(version, '7.2.51704') < 0)) OR NOT EXISTS (SELECT 1 FROM system_info WHERE cpu_type LIKE 'ARM%');"
)

// windowsFMAApp is a Windows catalog entry as apps.json has it, with the installer title
// agreeing with the catalog name. titleID doubles as the identity used for ambiguity checks.
func windowsFMAApp(titleID uint, name, uniqueIdentifier, patchQuery string) fleet.MaintainedApp {
	return fleet.MaintainedApp{
		Name: name, UniqueIdentifier: uniqueIdentifier, PatchQuery: patchQuery,
		Platform: "windows", TitleID: &titleID, TitleName: name,
	}
}

func TestMatchWindowsFMATitle(t *testing.T) {
	// win builds an FMA whose installer title agrees with the catalog name, which is
	// the normal case. titleID doubles as the identity used for ambiguity checks.
	win := func(titleID uint, name, uniqueIdentifier string) fleet.MaintainedApp {
		return fleet.MaintainedApp{
			Name: name, UniqueIdentifier: uniqueIdentifier,
			Platform: "windows", TitleID: &titleID, TitleName: name,
		}
	}

	granola := win(1, "Granola", "Granola")
	zoom := win(2, "Zoom", "Zoom")
	// osquery reports "CPUID CPU-Z ...", so only the identifier is a usable prefix.
	cpuz := win(3, "CPU-Z", "CPUID CPU-Z")
	// apps.json entries are append-only, so some identifiers carry a stale version;
	// only the name is a usable prefix here.
	notion := win(4, "Notion", "Notion 6.1.0")
	// Two apps reporting under the same identifier that own different titles.
	acmeReader := win(5, "Acme Reader", "Acme")
	acmeWriter := win(6, "Acme Writer", "Acme")
	// A more specific app whose prefix extends another's.
	fooBar := win(7, "Foo Bar", "Foo Bar")
	foo := win(8, "Foo", "Foo")
	// The catalog renamed this app; the installer's title keeps the older name, which
	// is what inventory still reports under and where software must land.
	renamed := fleet.MaintainedApp{
		Name: "Zoom Workplace", UniqueIdentifier: "Zoom",
		Platform: "windows", TitleID: new(uint(9)), TitleName: "Zoom",
	}
	// Two slugs of the same app resolving to one title: not ambiguous.
	dupTitleID := uint(10)
	sameTitleA := fleet.MaintainedApp{Name: "Dup A", UniqueIdentifier: "Dup", Platform: "windows", TitleID: &dupTitleID, TitleName: "Dup"}
	sameTitleB := fleet.MaintainedApp{Name: "Dup B", UniqueIdentifier: "Dup", Platform: "windows", TitleID: &dupTitleID, TitleName: "Dup"}

	cases := []struct {
		name        string
		reported    string
		fmas        []fleet.MaintainedApp
		wantTitle   string
		wantTitleID uint
		wantOK      bool
	}{
		{"version suffix", "Granola 7.373.2", []fleet.MaintainedApp{granola}, "Granola", 1, true},
		{"exact name", "Granola", []fleet.MaintainedApp{granola}, "Granola", 1, true},
		{"case insensitive", "granola 7.373.2", []fleet.MaintainedApp{granola}, "Granola", 1, true},
		{"no separator is not a match", "Zoombie 5.0", []fleet.MaintainedApp{zoom}, "", 0, false},
		{"prefix without space", "Granolabar 1.0", []fleet.MaintainedApp{granola}, "", 0, false},
		{"unrelated", "Firefox 141.0", []fleet.MaintainedApp{granola, zoom}, "", 0, false},
		{"matches via unique identifier", "CPUID CPU-Z 2.16", []fleet.MaintainedApp{cpuz}, "CPU-Z", 3, true},
		{"matches via name when identifier is stale", "Notion 7.2.0", []fleet.MaintainedApp{notion}, "Notion", 4, true},
		{"stale identifier still matches itself", "Notion 6.1.0 extra", []fleet.MaintainedApp{notion}, "Notion", 4, true},
		{"longest prefix wins", "Foo Bar 1.0", []fleet.MaintainedApp{foo, fooBar}, "Foo Bar", 7, true},
		{"longest prefix wins regardless of order", "Foo Bar 1.0", []fleet.MaintainedApp{fooBar, foo}, "Foo Bar", 7, true},
		{"shorter prefix still matches its own app", "Foo 1.0", []fleet.MaintainedApp{foo, fooBar}, "Foo", 8, true},
		{"ambiguous tie is no match", "Acme 3.0", []fleet.MaintainedApp{acmeReader, acmeWriter}, "", 0, false},
		{"ambiguous tie is no match, reversed", "Acme 3.0", []fleet.MaintainedApp{acmeWriter, acmeReader}, "", 0, false},
		{"same destination is not ambiguous", "Dup 1.0", []fleet.MaintainedApp{sameTitleA, sameTitleB}, "Dup", 10, true},
		{"no FMAs", "Granola 7.373.2", nil, "", 0, false},

		// Destination is the installer's title, never the current catalog name.
		{"renamed catalog: matches old name", "Zoom 6.1.0", []fleet.MaintainedApp{renamed}, "Zoom", 9, true},
		{"renamed catalog: matches new name", "Zoom Workplace 7.0", []fleet.MaintainedApp{renamed}, "Zoom", 9, true},

		// MySQL's utf8mb4_unicode_ci ignores Unicode format characters, so Go-side
		// matching must too or a name carrying one silently stops matching.
		{"reported name with RTL mark", "Granola\u200f 7.373.2", []fleet.MaintainedApp{granola}, "Granola", 1, true},
		{"zero-width joiner in reported name", "Gran\u200dola 7.373.2", []fleet.MaintainedApp{granola}, "Granola", 1, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := newWindowsFMAMatcher(c.fmas).match(c.reported, "", "")
			require.Equal(t, c.wantOK, ok)
			require.Equal(t, c.wantTitle, got.titleName)
			require.Equal(t, c.wantTitleID, got.titleID)
		})
	}
}

func TestMatchWindowsFMATitleCuratedRules(t *testing.T) {
	firefox := windowsFMAApp(1, "Mozilla Firefox", "Mozilla Firefox (x64 en-US)", firefoxPatchQuery)
	firefoxESR := windowsFMAApp(2, "Mozilla Firefox ESR", "Mozilla Firefox 140.7.1 ESR (x64 en-US)", firefoxESRPatchQuery)
	dotnet8 := windowsFMAApp(3, "Microsoft .NET Runtime 8", "Microsoft .NET Runtime - 8", dotnet8PatchQuery)
	python313 := windowsFMAApp(4, "Python 3.13", "Python 3.13.13 (64-bit)", python313PatchQuery)
	python314 := windowsFMAApp(5, "Python 3.14", "Python 3.14.5 (64-bit)", python314PatchQuery)
	qgis := windowsFMAApp(6, "QGIS", "QGIS", qgisPatchQuery)
	qgisLTR := windowsFMAApp(7, "QGIS LTR", "QGIS LTR", qgisLTRPatchQuery)
	realVNC := windowsFMAApp(8, "RealVNC Connect Viewer", "RealVNC Connect Viewer", realVNCPatchQuery)
	granola := windowsFMAApp(9, "Granola", "Granola", granolaPatchQuery)
	burpCommunity := windowsFMAApp(10, "Burp Suite Community Edition", "Burp Suite", burpPatchQuery)
	burpPro := windowsFMAApp(11, "Burp Suite Professional", "Burp Suite", burpPatchQuery)
	zoom := windowsFMAApp(12, "Zoom", "Zoom Workplace (X64)", zoomPatchQuery)
	zoomARM := windowsFMAApp(13, "Zoom (ARM64)", "Zoom Workplace (ARM64)", zoomARMPatchQuery)
	noPublisher := windowsFMAApp(14, "GoldenDict-ng", "GoldenDict-ng",
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'GoldenDict-ng%' AND version_compare(version, '25.6.0') < 0);")
	noName := windowsFMAApp(15, "Acme", "Acme",
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE publisher = 'Acme' AND version_compare(version, '1.0') < 0);")
	wildcardOnly := windowsFMAApp(17, "Bar", "Bar",
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE '%Bar%' AND publisher = 'Acme' AND version_compare(version, '1.0') < 0);")
	matchAllPublisher := windowsFMAApp(18, "Baz", "Baz",
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Baz%' AND publisher LIKE '%' AND version_compare(version, '1.0') < 0);")
	unparseable := windowsFMAApp(16, "Foo", "Foo",
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Foo%' AND publisher = 'Foo Inc' AND install_location LIKE '%Foo%' AND version_compare(version, '1.0') < 0);")

	cases := []struct {
		name        string
		reported    string
		vendor      string
		version     string
		fmas        []fleet.MaintainedApp
		wantTitleID uint
		wantOK      bool
	}{
		{"version mid-name goes to the app whose rule accepts it",
			"Mozilla Firefox 141.0 ESR (x64 en-US)", "Mozilla", "141.0", []fleet.MaintainedApp{firefox, firefoxESR}, 2, true},
		{"prefix still wins where no rule accepts",
			"Mozilla Firefox (x64 de)", "Mozilla", "141.0", []fleet.MaintainedApp{firefox, firefoxESR}, 1, true},
		{"two name clauses", "Microsoft .NET Runtime - 8.0.33 (x64)", "Microsoft Corporation", "8.0.33", []fleet.MaintainedApp{dotnet8}, 3, true},
		{"two name clauses, other architecture", "Microsoft .NET Runtime - 8.0.33 (arm64)", "Microsoft Corporation", "8.0.33", []fleet.MaintainedApp{dotnet8}, 0, false},
		{"version glued to the name", "Python 3.13.16 (64-bit)", "Python Software Foundation", "3.13.16150.0", []fleet.MaintainedApp{python313, python314}, 4, true},
		{"sibling series", "Python 3.14.1 (64-bit)", "Python Software Foundation", "3.14.1150.0", []fleet.MaintainedApp{python313, python314}, 5, true},
		{"version clause picks LTR", "QGIS 3.44.15 'Solothurn'", "QGIS.org", "3.44.15", []fleet.MaintainedApp{qgis, qgisLTR}, 7, true},
		{"version clause picks latest", "QGIS 4.2.3 'Wien'", "QGIS.org", "4.2.3", []fleet.MaintainedApp{qgis, qgisLTR}, 6, true},
		{"OR group", "RealVNC Viewer 8.6.2", "RealVNC", "8.6.2", []fleet.MaintainedApp{realVNC}, 8, true},
		{"other vendor falls back to the prefix", "Granola 7.373.2", "Someone Else", "7.373.2", []fleet.MaintainedApp{granola}, 9, true},
		{"rule match needs the vendor", "RealVNC Viewer 8.6.2", "Someone Else", "8.6.2", []fleet.MaintainedApp{realVNC}, 0, false},
		{"vendor compared case-insensitively", "RealVNC Viewer 8.6.2", "realvnc", "8.6.2", []fleet.MaintainedApp{realVNC}, 8, true},
		{"identical rules leave the prefix to decide", "Burp Suite Professional 2026.1", "PortSwigger Web Security", "2026.1", []fleet.MaintainedApp{burpCommunity, burpPro}, 11, true},
		{"identical rules and a prefix tie stay unmatched", "Burp Suite 2026.1", "PortSwigger Web Security", "2026.1", []fleet.MaintainedApp{burpCommunity, burpPro}, 0, false},
		{"a looser rule does not take a longer prefix match", "Zoom Workplace (ARM64) 7.2", "Zoom", "7.2", []fleet.MaintainedApp{zoom, zoomARM}, 13, true},
		{"rule without a publisher never accepts", "GoldenDict-ng1.0", "", "1.0", []fleet.MaintainedApp{noPublisher}, 0, false},
		{"rule without a name never accepts", "Anything 1.0", "Acme", "1.0", []fleet.MaintainedApp{noName}, 0, false},
		{"rule pinning no name text never accepts", "Foo Bar 1.0", "Acme", "1.0", []fleet.MaintainedApp{wildcardOnly}, 0, false},
		{"rule whose publisher pattern pins no text never accepts", "Bazooka 1.0", "Acme", "1.0", []fleet.MaintainedApp{matchAllPublisher}, 0, false},
		{"unparseable rule keeps prefix behavior", "Foobar 1.0", "Foo Inc", "1.0", []fleet.MaintainedApp{unparseable}, 0, false},
		{"unparseable rule keeps prefix behavior, positive", "Foo 1.0", "Foo Inc", "1.0", []fleet.MaintainedApp{unparseable}, 16, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := newWindowsFMAMatcher(c.fmas).match(c.reported, c.vendor, c.version)
			require.Equal(t, c.wantOK, ok)
			require.Equal(t, c.wantTitleID, got.titleID)
		})
	}

	// Ingestion falls back to an empty matcher when the app lookup fails.
	_, ok := (&windowsFMAMatcher{}).match("Granola 7.373.2", "Granola", "7.373.2")
	require.False(t, ok)
}

func TestWindowsFMARuleMayDecide(t *testing.T) {
	m := newWindowsFMAMatcher([]fleet.MaintainedApp{
		windowsFMAApp(1, "Mozilla Firefox", "Mozilla Firefox (x64 en-US)", firefoxPatchQuery),
		windowsFMAApp(2, "Mozilla Firefox ESR", "Mozilla Firefox 140.7.1 ESR (x64 en-US)", firefoxESRPatchQuery),
		windowsFMAApp(3, "Zoom", "Zoom Workplace (X64)", zoomPatchQuery),
		windowsFMAApp(4, "Zoom (ARM64)", "Zoom Workplace (ARM64)", zoomARMPatchQuery),
		windowsFMAApp(5, "Python 3.13", "Python 3.13.13 (64-bit)", python313PatchQuery),
	})

	cases := []struct {
		name string
		want bool
	}{
		{"Mozilla Firefox 141.0 ESR (x64 en-US)", true},
		{"Python 3.13.16 (64-bit)", true},
		{"Zoom Workplace 7.2", true},
		// The Zoom rule's literal is shorter than the ARM64 prefix, so it can't win.
		{"Zoom Workplace (ARM64) 7.2", false},
		{"Python 3.12.1 (64-bit)", false},
		{"Mozilla Thunderbird 140.0", false},
	}
	for _, c := range cases {
		normalized := normalizeForMatch(c.name)
		_, prefixLen, _ := m.matchPrefix(normalized)
		require.Equal(t, c.want, m.ruleMayDecide(normalized, prefixLen), c.name)
	}
}

func TestWindowsFMALike(t *testing.T) {
	cases := []struct {
		s, pattern string
		want       bool
	}{
		{"abc", "abc", true},
		{"abc", "ab", false},
		{"abc", "a%", true},
		{"abc", "%c", true},
		{"abc", "%b%", true},
		{"abc", "a_c", true},
		{"ac", "a_c", false},
		{"", "%", true},
		{"", "_", false},
		{"a%c", "a%c", true},
		{"mozilla firefox 141.0 esr (x64 en-us)", "mozilla firefox%esr%", true},
		{"mozilla firefox 141.0 (x64 en-us)", "mozilla firefox%esr%", false},
		{"aaab", "%a%ab", true},
		{"日本語 1.0", "日本_ %", true},
		{"日本語", "%_語", true},
		{"ab", "__", true},
		{"a", "__", false},
		{"abc", "a%%c", true},
		{"ab", "abc", false},
		{"ab", "ab%_", false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, windowsFMALike(c.s, c.pattern), "%q LIKE %q", c.s, c.pattern)
	}
}
