package ai_tools

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/browserext"
)

func TestBrowserExtTypeRegistered(t *testing.T) {
	found := false
	for _, k := range allTypes {
		if k == "browser_extension" {
			found = true
		}
	}
	if !found {
		t.Fatal("browser_extension missing from allTypes")
	}
}

func TestBrowserExtRow(t *testing.T) {
	e := browserext.Extension{
		Browser: "brave", Engine: "chromium", Profile: "Default",
		ID: "abcID", Name: "Claude for Chrome", Version: "1.0.0",
		Path: "/x/manifest.json", Category: "ai-assistant",
		ManifestVer: 3, HostPerms: []string{"<all_urls>"},
		FromWebstore: 0, SignedState: -99, Sideloaded: true,
		SHA256: "deadbeef", RiskFlags: "broad_host_permissions,sideloaded_unverified",
		UID: "501", Username: "tester",
	}
	r := browserExtRow(e)

	if r["type"] != "browser_extension" || r["name"] != "Claude for Chrome" ||
		r["identifier"] != "abcID" || r["source"] != "brave" ||
		r["category"] != "ai-assistant" || r["location"] != "local" ||
		r["risk_flags"] != "broad_host_permissions,sideloaded_unverified" ||
		r["sha256"] != "deadbeef" || r["username"] != "tester" {
		t.Errorf("row columns wrong: %+v", r)
	}

	var detail map[string]string
	if err := json.Unmarshal([]byte(r["detail"]), &detail); err != nil {
		t.Fatalf("detail not valid JSON: %v (%q)", err, r["detail"])
	}
	if detail["engine"] != "chromium" || detail["profile"] != "Default" ||
		detail["browser"] != "brave" || detail["from_webstore"] != "false" {
		t.Errorf("detail wrong: %+v", detail)
	}
}
