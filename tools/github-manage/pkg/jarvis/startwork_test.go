package jarvis

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		in     string
		maxLen int
		want   string
	}{
		{"Add VC++ Redistributable (x64)", 40, "add-vc-redistributable-x64"},
		{"  Trim  Spaces  ", 40, "trim-spaces"},
		{"a very long title that should be truncated on a hyphen boundary here", 20, "a-very-long-title"},
		{"UPPER case", 40, "upper-case"},
	}
	for _, tt := range tests {
		if got := slugify(tt.in, tt.maxLen); got != tt.want {
			t.Errorf("slugify(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
		}
	}
}

func TestSuggestBranch(t *testing.T) {
	got := suggestBranch("GeorgeKarr", 38348, "Do the thing")
	want := "georgekarr-38348-do-the-thing"
	if got != want {
		t.Errorf("suggestBranch = %q, want %q", got, want)
	}
	if got := suggestBranch("", 100, "no login"); got != "100-no-login" {
		t.Errorf("suggestBranch without login = %q", got)
	}
}

func TestNormalizeRemote(t *testing.T) {
	tests := []struct {
		url  string
		want string
		ok   bool
	}{
		{"git@github.com:fleetdm/fleet.git", "fleetdm/fleet", true},
		{"https://github.com/fleetdm/fleet.git", "fleetdm/fleet", true},
		{"https://github.com/fleetdm/fleet", "fleetdm/fleet", true},
		{"ssh://example.com/other.git", "", false},
	}
	for _, tt := range tests {
		got, ok := normalizeRemote(tt.url)
		if ok != tt.ok || got != tt.want {
			t.Errorf("normalizeRemote(%q) = %q,%v want %q,%v", tt.url, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFocusStoreToggle(t *testing.T) {
	s, _ := LoadFocusStore("")
	if on := s.Toggle(42); !on || !s.Has(42) {
		t.Fatalf("Toggle should pin 42")
	}
	if on := s.Toggle(42); on || s.Has(42) {
		t.Fatalf("Toggle should unpin 42")
	}
}
