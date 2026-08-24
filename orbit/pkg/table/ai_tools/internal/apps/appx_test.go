package apps

import (
	"path/filepath"
	"testing"
)

func TestParsePackageFullName(t *testing.T) {
	cases := []struct {
		pfn        string
		wantOK     bool
		name       string
		version    string
		resourceID string
	}{
		{
			pfn: "OpenAI.ChatGPT-Desktop_1.2024.30.0_x64__2p2nqsd0c76g0", wantOK: true,
			name: "OpenAI.ChatGPT-Desktop", version: "1.2024.30.0",
		},
		{
			pfn: "ElementLabs.LMStudio_0.3.9.0_x64__k1h2f0dnjkqp0", wantOK: true,
			name: "ElementLabs.LMStudio", version: "0.3.9.0",
		},
		{
			// Observed on a Windows 11 ARM64 host.
			pfn: "OpenAI.ChatGPT-Desktop_1.2026.190.0_arm64__2p2nqsd0c76g0", wantOK: true,
			name: "OpenAI.ChatGPT-Desktop", version: "1.2026.190.0",
		},
		{
			// A package name may itself contain dots; only underscores delimit fields.
			pfn: "Microsoft.VCLibs.140.00_14.0.30704.0_x64__8wekyb3d8bbwe", wantOK: true,
			name: "Microsoft.VCLibs.140.00", version: "14.0.30704.0",
		},
		{
			// Resource ("split") satellite packages carry the same identity as the
			// main package but a non-empty resource id.
			pfn: "Microsoft.WindowsCalculator_11.2210.0.0_neutral_split.scale-100_8wekyb3d8bbwe", wantOK: true,
			name: "Microsoft.WindowsCalculator", version: "11.2210.0.0", resourceID: "split.scale-100",
		},

		// Malformed names must be rejected rather than yielding a garbage row.
		{pfn: "", wantOK: false},
		{pfn: "NotAPackage", wantOK: false},
		{pfn: "A_B_C_D_E", wantOK: false},                   // version is not numeric
		{pfn: "Foo_1.0.0.0_x64_", wantOK: false},            // truncated: no publisher id
		{pfn: "Foo__x64__8wekyb3d8bbwe", wantOK: false},     // empty version
		{pfn: "_1.0.0.0_x64__8wekyb3d8bbwe", wantOK: false}, // empty name
	}

	for _, c := range cases {
		got, ok := parsePackageFullName(c.pfn)
		if ok != c.wantOK {
			t.Errorf("parsePackageFullName(%q) ok=%v want %v", c.pfn, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if got.Name != c.name || got.Version != c.version || got.ResourceID != c.resourceID {
			t.Errorf("parsePackageFullName(%q) = %+v want name=%q version=%q resourceID=%q",
				c.pfn, got, c.name, c.version, c.resourceID)
		}
	}
}

func TestAppxPackageIsResourcePackage(t *testing.T) {
	cases := []struct {
		resourceID string
		want       bool
	}{
		{"", false},
		{"split.scale-100", true},
		{"split.language-en", true},
	}
	for _, c := range cases {
		p := appxPackage{Name: "Some.App", Version: "1.0.0.0", ResourceID: c.resourceID}
		if got := p.isResourcePackage(); got != c.want {
			t.Errorf("appxPackage{ResourceID:%q}.isResourcePackage() = %v want %v", c.resourceID, got, c.want)
		}
	}
}

// TestAppxPackageFamilyName covers the identifier used to attribute an install
// to a user: %LOCALAPPDATA%\Packages subdirectories are named by family name,
// not by package full name.
func TestAppxPackageFamilyName(t *testing.T) {
	cases := []struct{ pfn, want string }{
		{"OpenAI.ChatGPT-Desktop_1.2026.190.0_arm64__2p2nqsd0c76g0", "OpenAI.ChatGPT-Desktop_2p2nqsd0c76g0"},
		{"Microsoft.WindowsCalculator_11.2210.0.0_x64__8wekyb3d8bbwe", "Microsoft.WindowsCalculator_8wekyb3d8bbwe"},
	}
	for _, c := range cases {
		p, ok := parsePackageFullName(c.pfn)
		if !ok {
			t.Fatalf("parsePackageFullName(%q) failed", c.pfn)
		}
		if got := p.FamilyName(); got != c.want {
			t.Errorf("FamilyName(%q) = %q want %q", c.pfn, got, c.want)
		}
	}
	if got := (appxPackage{}).FamilyName(); got != "" {
		t.Errorf("zero appxPackage FamilyName() = %q want \"\"", got)
	}
}

// TestAppxInstallRootFrom covers the WOW64 trap: %ProgramFiles% is redirected to
// "Program Files (x86)" for a 32-bit process, and that directory holds no
// WindowsApps, so %ProgramW6432% must win whenever it is set.
func TestAppxInstallRootFrom(t *testing.T) {
	cases := []struct {
		name                                 string
		programW6432, programFiles, sysDrive string
		want                                 string
	}{
		{
			name:         "64-bit process: both set and identical",
			programW6432: `C:\Program Files`, programFiles: `C:\Program Files`,
			want: filepath.Join(`C:\Program Files`, "WindowsApps"),
		},
		{
			name:         "32-bit process: ProgramFiles is redirected and must lose",
			programW6432: `C:\Program Files`, programFiles: `C:\Program Files (x86)`,
			want: filepath.Join(`C:\Program Files`, "WindowsApps"),
		},
		{
			name:         "32-bit Windows: ProgramW6432 unset, ProgramFiles is correct",
			programFiles: `C:\Program Files`,
			want:         filepath.Join(`C:\Program Files`, "WindowsApps"),
		},
		{
			name:     "no Program Files vars: fall back to the system drive",
			sysDrive: "D:",
			want:     filepath.Join(filepath.Join(`D:\`, "Program Files"), "WindowsApps"),
		},
		{
			name: "nothing set at all",
			want: filepath.Join(filepath.Join(`C:\`, "Program Files"), "WindowsApps"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := appxInstallRootFrom(c.programW6432, c.programFiles, c.sysDrive); got != c.want {
				t.Errorf("appxInstallRootFrom(%q, %q, %q) = %q want %q",
					c.programW6432, c.programFiles, c.sysDrive, got, c.want)
			}
		})
	}
}

func TestAppxLiteral(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ChatGPT", "ChatGPT"},
		{"LM Studio", "LM Studio"},
		{"", ""},
		// MUI indirect strings are unresolvable without SHLoadIndirectString and
		// are useless for token matching, so they are discarded.
		{"@{OpenAI.ChatGPT_1.0.0.0_x64__2p2nqsd0c76g0?ms-resource://OpenAI.ChatGPT/Resources/AppName}", ""},
		{"@{Microsoft.WindowsCalculator?ms-resource://Foo}", ""},
		{"@", ""},
	}
	for _, c := range cases {
		if got := appxLiteral(c.in); got != c.want {
			t.Errorf("appxLiteral(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestAppxVendor(t *testing.T) {
	cases := []struct {
		display string
		dn      string
		want    string
	}{
		// A literal PublisherDisplayName always wins.
		{"OpenAI", "CN=OpenAI, O=OpenAI OpCo, LLC, C=US", "OpenAI"},

		// When it is an indirect string, fall back to the certificate common name.
		{"@{Foo?ms-resource://Foo/PublisherDisplayName}", "CN=OpenAI, O=OpenAI OpCo, LLC, L=San Francisco, S=California, C=US", "OpenAI"},
		{"", "CN=OpenAI, O=OpenAI OpCo, LLC, L=San Francisco, S=California, C=US", "OpenAI"},

		// A comma inside the common name must not truncate it: only a comma that
		// starts the next attribute (", O=") ends the value.
		{"", "CN=Element Labs, Inc., O=Element Labs, C=US", "Element Labs, Inc."},
		{"", "CN=Solo", "Solo"},

		// Nothing usable.
		{"", "O=NoCommonName, C=US", ""},
		{"", "", ""},
		{"@{Foo?ms-resource://x}", "", ""},
	}
	for _, c := range cases {
		if got := appxVendor(c.display, c.dn); got != c.want {
			t.Errorf("appxVendor(%q, %q) = %q want %q", c.display, c.dn, got, c.want)
		}
	}
}

// TestMatchKnownPackageNames locks in that MSIX package identity names match the
// existing knownApps tokens on their own, so the collector does not need to
// space-normalize them. That matters: normalizing "NVIDIA.NVIDIAControlPanel"
// into "nvidia nvidiacontrolpanel" makes it match the "dia " token for the Dia
// browser, and dotted package names avoid that collision entirely.
func TestMatchKnownPackageNames(t *testing.T) {
	cases := []struct {
		pkgName string
		wantOK  bool
		want    string
	}{
		{"OpenAI.ChatGPT-Desktop", true, "chatgpt"},
		{"ElementLabs.LMStudio", true, "lm-studio"},
		// The "." delimits a word, so the "comet" token sees the product segment
		// and wins over the publisher-only "perplexity" match.
		{"Perplexity.Comet", true, "comet"},

		{"Microsoft.WindowsCalculator", false, ""},
		{"Microsoft.VCLibs.140.00", false, ""},
		{"NVIDIA.NVIDIAControlPanel", false, ""},
	}
	for _, c := range cases {
		k, ok := matchKnown(c.pkgName)
		if ok != c.wantOK {
			t.Errorf("matchKnown(%q) ok=%v want %v (matched %q)", c.pkgName, ok, c.wantOK, k.name)
			continue
		}
		if ok && k.name != c.want {
			t.Errorf("matchKnown(%q) = %q want %q", c.pkgName, k.name, c.want)
		}
	}
}
