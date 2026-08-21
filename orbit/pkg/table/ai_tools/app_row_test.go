package ai_tools

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/apps"
)

// TestAppRow locks in the apps column semantics: `name` carries the installed
// program's real display name and `identifier` carries the known-app key, so a
// wrong match shows the real program instead of masquerading as the known app;
// the bundle id lives in `detail`.
func TestAppRow(t *testing.T) {
	a := apps.App{
		Name:           "claude-desktop", // known-app key
		DisplayName:    "Claude",
		Vendor:         "Anthropic",
		BundleID:       "com.anthropic.claude",
		Version:        "1.2.3",
		Path:           "/Applications/Claude.app",
		PlatformSource: "applications",
		Scope:          "system",
		Running:        1,
		PID:            42,
		SHA256:         "deadbeef",
	}
	r := appRow(a)

	if r["type"] != "apps" || r["name"] != "Claude" ||
		r["identifier"] != "claude-desktop" ||
		r["location"] != "local" || r["source"] != "applications" ||
		r["version"] != "1.2.3" || r["path"] != "/Applications/Claude.app" ||
		r["running"] != "1" || r["pid"] != "42" || r["sha256"] != "deadbeef" {
		t.Errorf("row columns wrong: %+v", r)
	}

	var detail map[string]string
	if err := json.Unmarshal([]byte(r["detail"]), &detail); err != nil {
		t.Fatalf("detail not valid JSON: %v (%q)", err, r["detail"])
	}
	if detail["vendor"] != "Anthropic" || detail["bundle_id"] != "com.anthropic.claude" ||
		detail["scope"] != "system" {
		t.Errorf("detail wrong: %+v", detail)
	}
}
