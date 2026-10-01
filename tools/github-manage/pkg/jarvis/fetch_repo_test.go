package jarvis

import "testing"

func TestRepoFromURL(t *testing.T) {
	cases := []struct{ url, want string }{
		{"https://github.com/fleetdm/confidential/issues/17176", "fleetdm/confidential"},
		{"https://github.com/fleetdm/fleet/issues/123", "fleetdm/fleet"},
		{"https://github.com/fleetdm/fleet", "fleetdm/fleet"},
		{"https://example.com/nope", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := repoFromURL(c.url); got != c.want {
			t.Errorf("repoFromURL(%q) = %q, want %q", c.url, got, c.want)
		}
	}
	if got := repoOr("https://github.com/fleetdm/confidential/issues/1", "fleetdm/fleet"); got != "fleetdm/confidential" {
		t.Errorf("repoOr should prefer the URL's repo, got %q", got)
	}
	if got := repoOr("not-a-url", "fleetdm/fleet"); got != "fleetdm/fleet" {
		t.Errorf("repoOr fallback: got %q", got)
	}
}
