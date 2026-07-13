package browserext

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

func TestHasBroadHostPerms(t *testing.T) {
	yes := [][]string{
		{"<all_urls>"},
		{"tabs", "*://*/*"},
		{"https://*/*"},
		{"http://*/*"},
		{" <all_urls> "},
	}
	for _, p := range yes {
		if !hasBroadHostPerms(p) {
			t.Errorf("hasBroadHostPerms(%v) = false, want true", p)
		}
	}
	no := [][]string{
		{"tabs", "storage"},
		{"https://mail.google.com/*"},
		nil,
	}
	for _, p := range no {
		if hasBroadHostPerms(p) {
			t.Errorf("hasBroadHostPerms(%v) = true, want false", p)
		}
	}
}

func TestChromiumSideloaded(t *testing.T) {
	// fromWebstore: -1 unknown, 0 no, 1 yes. location: 0 unknown, 1 internal, 4 unpacked, 5 component, 10 external.
	cases := []struct {
		fw, loc int
		want    bool
	}{
		{1, 1, false},  // store + internal
		{0, 1, true},   // explicitly not webstore
		{-1, 4, true},  // unpacked
		{-1, 10, true}, // external/policy
		{-1, 5, false}, // component
		{-1, 0, false}, // both unknown -> conservative, no flag
		{1, 4, true},   // store-flagged but unpacked location -> still anomalous
		{1, 10, true},  // store-flagged but external/policy location -> still anomalous
	}
	for _, c := range cases {
		if got := chromiumSideloaded(c.fw, c.loc); got != c.want {
			t.Errorf("chromiumSideloaded(%d,%d)=%v want %v", c.fw, c.loc, got, c.want)
		}
	}
}

func TestGeckoSideloaded(t *testing.T) {
	cases := []struct {
		signed  int
		foreign bool
		want    bool
	}{
		{2, false, false},                  // privileged
		{1, false, false},                  // signed
		{0, false, true},                   // missing signature
		{-1, false, true},                  // unknown-signature state
		{1, true, true},                    // signed but foreign-installed
		{signedStateUnknown, false, false}, // truly unknown -> conservative
	}
	for _, c := range cases {
		if got := geckoSideloaded(c.signed, c.foreign); got != c.want {
			t.Errorf("geckoSideloaded(%d,%v)=%v want %v", c.signed, c.foreign, got, c.want)
		}
	}
}

func TestComputeRisk(t *testing.T) {
	e := Extension{HostPerms: []string{"<all_urls>"}, Sideloaded: true}
	e.computeRisk()
	if e.RiskFlags != "broad_host_permissions,sideloaded_unverified" {
		t.Errorf("RiskFlags=%q want both flags in order", e.RiskFlags)
	}
	clean := Extension{HostPerms: []string{"tabs"}, Sideloaded: false}
	clean.computeRisk()
	if clean.RiskFlags != "" {
		t.Errorf("RiskFlags=%q want empty", clean.RiskFlags)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectChromiumProfile(t *testing.T) {
	home := t.TempDir()
	profile := t.TempDir()
	exts := filepath.Join(profile, "Extensions")

	// AI extension on disk: i18n name, broad host perms.
	aiID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	writeFile(t, filepath.Join(exts, aiID, "1.0.0", "manifest.json"),
		`{"name":"__MSG_extName__","version":"1.0.0","default_locale":"en","manifest_version":3,"host_permissions":["<all_urls>"]}`)
	writeFile(t, filepath.Join(exts, aiID, "1.0.0", "_locales", "en", "messages.json"),
		`{"extName":{"message":"ChatGPT Sidebar"}}`)

	// Non-AI extension on disk -> must be dropped.
	nonAI := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	writeFile(t, filepath.Join(exts, nonAI, "2.0.0", "manifest.json"),
		`{"name":"Prettier","version":"2.0.0","manifest_version":3}`)

	// Unpacked AI extension: NOT under Extensions/, only in Preferences (location
	// 4). Its source lives under the user's home, as a real unpacked extension
	// does — collectChromiumProfile only hashes a Preferences "path" contained in
	// the owning home (attacker-controlled paths outside it are refused).
	unpackedSrc := filepath.Join(home, "unpacked-ext")
	writeFile(t, filepath.Join(unpackedSrc, "manifest.json"),
		`{"name":"Claude Dev","version":"9.9.9","manifest_version":3,"permissions":["<all_urls>"]}`)
	unpackedID := "cccccccccccccccccccccccccccccccc"

	// Secure Preferences: AI ext not-from-webstore (sideloaded), unpacked entry.
	prefs := map[string]any{
		"extensions": map[string]any{
			"settings": map[string]any{
				aiID:       map[string]any{"from_webstore": false, "location": 1},
				unpackedID: map[string]any{"location": 4, "path": unpackedSrc, "manifest": map[string]any{"name": "Claude Dev", "version": "9.9.9", "permissions": []string{"<all_urls>"}}},
			},
		},
	}
	pb, _ := json.Marshal(prefs)
	writeFile(t, filepath.Join(profile, "Secure Preferences"), string(pb))

	got := collectChromiumProfile(profile, "chrome", "Default", homes.Home{UID: "501", Username: "tester", Dir: home})

	by := map[string]Extension{}
	for _, e := range got {
		by[e.ID] = e
	}
	if _, ok := by[nonAI]; ok {
		t.Error("non-AI Prettier should be dropped (AI-only table)")
	}
	ai, ok := by[aiID]
	if !ok {
		t.Fatalf("AI extension not found; got %d (%+v)", len(got), got)
	}
	if ai.Name != "ChatGPT Sidebar" {
		t.Errorf("name=%q want resolved i18n 'ChatGPT Sidebar'", ai.Name)
	}
	if ai.Engine != "chromium" || ai.Browser != "chrome" || ai.Profile != "Default" {
		t.Errorf("metadata wrong: %+v", ai)
	}
	if ai.SHA256 == "" {
		t.Error("AI extension manifest hash empty")
	}
	for _, want := range []string{"broad_host_permissions", "sideloaded_unverified"} {
		if !contains(ai.RiskFlags, want) {
			t.Errorf("RiskFlags=%q missing %q", ai.RiskFlags, want)
		}
	}
	up, ok := by[unpackedID]
	if !ok {
		t.Fatal("unpacked extension (Preferences-only) not recovered")
	}
	if up.Name != "Claude Dev" || up.SHA256 == "" || !contains(up.RiskFlags, "sideloaded_unverified") {
		t.Errorf("unpacked ext wrong: %+v", up)
	}
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

// TestChromiumRefusesOutOfHomePath verifies that a Chromium Preferences "path"
// pointing outside the owning home is not hashed: the row is still recovered,
// but its SHA256 is empty so the root scanner is never steered at an arbitrary
// absolute path from the user-writable Preferences file.
func TestChromiumRefusesOutOfHomePath(t *testing.T) {
	home := t.TempDir()
	profile := t.TempDir()
	outside := t.TempDir() // NOT under home
	writeFile(t, filepath.Join(outside, "manifest.json"),
		`{"name":"Claude Dev","version":"1.0","manifest_version":3}`)

	id := "cccccccccccccccccccccccccccccccc"
	prefs := map[string]any{
		"extensions": map[string]any{
			"settings": map[string]any{
				id: map[string]any{"location": 4, "path": outside, "manifest": map[string]any{"name": "Claude Dev", "version": "1.0"}},
			},
		},
	}
	pb, _ := json.Marshal(prefs)
	writeFile(t, filepath.Join(profile, "Secure Preferences"), string(pb))

	got := collectChromiumProfile(profile, "chrome", "Default", homes.Home{UID: "501", Username: "tester", Dir: home})

	var ext *Extension
	for i := range got {
		if got[i].ID == id {
			ext = &got[i]
		}
	}
	if ext == nil {
		t.Fatal("out-of-home extension should still be recovered from Preferences")
	}
	if ext.SHA256 != "" {
		t.Errorf("SHA256 = %q, want empty (out-of-home path must not be hashed)", ext.SHA256)
	}
}

// TestGeckoRejectsTraversalID verifies an addon id from the user-writable
// extensions.json that contains path-traversal characters is dropped, so the
// root scanner cannot be steered outside the profile's extensions dir.
func TestGeckoRejectsTraversalID(t *testing.T) {
	profile := t.TempDir()
	writeFile(t, filepath.Join(profile, "extensions.json"), `{"addons":[
		{"id":"../../../../etc/evil","type":"extension","location":"app-profile","version":"1.0","defaultLocale":{"name":"ChatGPT"}},
		{"id":"chatgpt@ai","type":"extension","location":"app-profile","version":"1.0","defaultLocale":{"name":"ChatGPT"}}
	]}`)

	got := collectGeckoProfile(profile, "firefox", "default", homes.Home{UID: "501", Username: "tester"})

	var cleanFound bool
	for _, e := range got {
		if strings.Contains(e.ID, "..") || strings.ContainsAny(e.ID, `/\`) {
			t.Errorf("traversal addon id surfaced: %q", e.ID)
		}
		if e.ID == "chatgpt@ai" {
			cleanFound = true
		}
	}
	if !cleanFound {
		t.Error("clean AI addon should still be surfaced")
	}
}

func writeXPI(t *testing.T, path, manifestJSON string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(manifestJSON)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectGeckoProfile(t *testing.T) {
	profile := t.TempDir()
	aiID := "ai-helper@example.com"

	extJSON := `{"addons":[
	  {"id":"ai-helper@example.com","type":"extension","version":"1.2.0","location":"app-profile","signedState":0,"foreignInstall":false,"defaultLocale":{"name":"Perplexity Helper"},"userPermissions":{"origins":["<all_urls>"]}},
	  {"id":"darktheme@example.com","type":"theme","version":"1.0","location":"app-profile","defaultLocale":{"name":"Dark AI Theme"}},
	  {"id":"builtin@mozilla.org","type":"extension","version":"1.0","location":"app-system-defaults","defaultLocale":{"name":"Copilot Builtin"}}
	]}`
	writeFile(t, filepath.Join(profile, "extensions.json"), extJSON)
	writeXPI(t, filepath.Join(profile, "extensions", aiID+".xpi"),
		`{"name":"Perplexity Helper","version":"1.2.0","host_permissions":["<all_urls>"]}`)

	got := collectGeckoProfile(profile, "firefox", "default-release", homes.Home{Username: "tester"})

	if len(got) != 1 {
		t.Fatalf("got %d extensions want 1 (theme + builtin must be skipped): %+v", len(got), got)
	}
	e := got[0]
	if e.ID != aiID || e.Engine != "gecko" || e.Browser != "firefox" {
		t.Errorf("metadata wrong: %+v", e)
	}
	if e.SHA256 == "" {
		t.Error("xpi hash empty")
	}
	for _, want := range []string{"broad_host_permissions", "sideloaded_unverified"} {
		if !contains(e.RiskFlags, want) {
			t.Errorf("RiskFlags=%q missing %q", e.RiskFlags, want)
		}
	}
}

func TestGeckoXPIHostPermFallback(t *testing.T) {
	profile := t.TempDir()
	addonID := "claude-for-firefox@example.com"

	// extensions.json: one extension, NO userPermissions block (origins will be empty).
	// signedState:1 means signed → NOT sideloaded.
	extJSON := `{"addons":[
	  {"id":"claude-for-firefox@example.com","type":"extension","version":"1.0.0","location":"app-profile","signedState":1,"foreignInstall":false,"defaultLocale":{"name":"Claude for Firefox"}}
	]}`
	writeFile(t, filepath.Join(profile, "extensions.json"), extJSON)

	// XPI whose manifest.json carries host_permissions — the fallback path reads this.
	writeXPI(t, filepath.Join(profile, "extensions", addonID+".xpi"),
		`{"name":"Claude for Firefox","version":"1.0.0","host_permissions":["<all_urls>"]}`)

	got := collectGeckoProfile(profile, "firefox", "default", homes.Home{Username: "t"})

	if len(got) != 1 {
		t.Fatalf("got %d extensions want 1: %+v", len(got), got)
	}
	e := got[0]
	if !contains(e.RiskFlags, "broad_host_permissions") {
		t.Errorf("RiskFlags=%q missing broad_host_permissions (xpi fallback not read)", e.RiskFlags)
	}
	if contains(e.RiskFlags, "sideloaded_unverified") {
		t.Errorf("RiskFlags=%q contains sideloaded_unverified but extension is signed", e.RiskFlags)
	}
}

func TestGeckoProfilesINI(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Profiles", "abcd.default-release", "extensions.json"), `{"addons":[]}`)
	writeFile(t, filepath.Join(root, "profiles.ini"),
		"[Profile0]\nName=default-release\nIsRelative=1\nPath=Profiles/abcd.default-release\nDefault=1\n\n[General]\nVersion=2\n")

	profs := geckoProfiles(root)
	if len(profs) != 1 {
		t.Fatalf("geckoProfiles found %d want 1: %+v", len(profs), profs)
	}
	if filepath.Base(profs[0].path) != "abcd.default-release" {
		t.Errorf("profile path=%q want .../abcd.default-release", profs[0].path)
	}
}

func TestScan(t *testing.T) {
	home := t.TempDir()
	r := paths.For(home)

	// Drop a Chrome profile fixture at the path the package itself computes.
	var chromeRoot string
	for _, br := range chromiumRoots(r) {
		if br.label == "chrome" {
			chromeRoot = br.dir
		}
	}
	if chromeRoot == "" {
		t.Fatal("chromiumRoots produced no chrome entry for this OS")
	}
	profile := filepath.Join(chromeRoot, "Default")
	writeFile(t, filepath.Join(profile, "Extensions", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "1.0.0", "manifest.json"),
		`{"name":"Claude for Chrome","version":"1.0.0","manifest_version":3,"host_permissions":["<all_urls>"]}`)
	writeFile(t, filepath.Join(profile, "Preferences"),
		`{"extensions":{"settings":{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":{"from_webstore":true,"location":1}}}}`)

	// Drop a Firefox profile fixture similarly.
	var ffRoot string
	for _, br := range geckoRoots(r) {
		if br.label == "firefox" {
			ffRoot = br.dir
		}
	}
	if ffRoot == "" {
		t.Fatal("geckoRoots produced no firefox entry for this OS")
	}
	ffProfile := filepath.Join(ffRoot, "Profiles", "xxxx.default")
	writeFile(t, filepath.Join(ffProfile, "extensions.json"),
		`{"addons":[{"id":"copilot@x","type":"extension","version":"1.0","location":"app-profile","signedState":1,"defaultLocale":{"name":"Copilot"},"userPermissions":{"origins":["https://github.com/*"]}}]}`)
	writeFile(t, filepath.Join(ffRoot, "profiles.ini"),
		"[Profile0]\nIsRelative=1\nPath=Profiles/xxxx.default\n")

	got := Scan(homes.Home{Dir: home, UID: "501", Username: "tester"})

	var sawChrome, sawFirefox bool
	for _, e := range got {
		if e.Engine == "chromium" && e.Browser == "chrome" {
			sawChrome = true
			if e.Username != "tester" {
				t.Errorf("ownership not stamped: %+v", e)
			}
		}
		if e.Engine == "gecko" && e.Browser == "firefox" {
			sawFirefox = true
			if e.Username != "tester" {
				t.Errorf("gecko ownership not stamped: %+v", e)
			}
		}
	}
	if !sawChrome {
		t.Errorf("Scan missed the chrome extension; got %+v", got)
	}
	if !sawFirefox {
		t.Errorf("Scan missed the firefox extension; got %+v", got)
	}
}
