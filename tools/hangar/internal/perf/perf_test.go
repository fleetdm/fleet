package perf

import (
	"encoding/json"
	"testing"
)

// Pin the catalog so a careless edit can't silently change which templates
// the UI offers. If you intentionally add/remove a template, update this.
func TestTemplatesCatalog(t *testing.T) {
	got := Templates()
	if len(got) != 12 {
		t.Fatalf("template count = %d, want 12", len(got))
	}

	wantIDs := []string{
		"macos_13.6.2", "macos_14.1.2",
		"windows_11", "windows_11_22H2_2861", "windows_11_22H2_3007",
		"ubuntu_22.04", "rhel_8", "rhel_9", "rhel_10",
		"iphone_14.6", "iphone_17", "ipad_13.18",
	}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Errorf("template[%d].ID = %q, want %q", i, got[i].ID, id)
		}
	}

	// mobile and apple flags drive UI gating — verify a representative set.
	byID := map[string]Template{}
	for _, tmpl := range got {
		byID[tmpl.ID] = tmpl
	}
	if m := byID["macos_14.1.2"]; m.Mobile || !m.Apple {
		t.Errorf("macos_14.1.2: mobile=%v apple=%v, want false/true", m.Mobile, m.Apple)
	}
	if m := byID["iphone_17"]; !m.Mobile || !m.Apple {
		t.Errorf("iphone_17: mobile=%v apple=%v, want true/true", m.Mobile, m.Apple)
	}
	if m := byID["windows_11"]; m.Mobile || m.Apple {
		t.Errorf("windows_11: mobile=%v apple=%v, want false/false", m.Mobile, m.Apple)
	}
}

// The frontend's PerfTemplate interface uses these exact lowercase keys.
func TestTemplateJSONKeys(t *testing.T) {
	b, err := json.Marshal(Templates()[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "label", "version", "mobile", "apple"} {
		if _, ok := m[key]; !ok {
			t.Errorf("serialized template missing key %q; got %v", key, m)
		}
	}
}
