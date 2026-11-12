package themes

import (
	"strings"
	"testing"
)

func TestAllThemesRegistered(t *testing.T) {
	want := []string{
		"hitchhikers", "goodplace", "parksrec", "tng", "lotr",
		"dbz", "robin_williams", "ghibli", "cosmere", "sailor_moon",
	}
	for _, name := range want {
		if _, err := Get(name); err != nil {
			t.Errorf("theme %q not registered: %v", name, err)
		}
	}
}

func TestMixIsNonEmpty(t *testing.T) {
	m := Mix()
	if len(m.Users) == 0 || len(m.Teams) == 0 || len(m.Policies) == 0 {
		t.Fatalf("mix theme came back empty: %+v", m)
	}
}

func TestEmailWrapsWithoutCollision(t *testing.T) {
	th, err := Get("hitchhikers")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	n := len(th.Users) * 3
	for i := 0; i < n; i++ {
		e := Email(th, i)
		if seen[e] {
			t.Errorf("duplicate email at i=%d: %s", i, e)
		}
		seen[e] = true
		if !strings.Contains(e, "@") {
			t.Errorf("not an email: %s", e)
		}
	}
}

func TestPickReturnsKindAwareDefault(t *testing.T) {
	th, _ := Get("hitchhikers")
	if got := Pick(th, "policy", 0); got.Name == "" {
		t.Fatalf("Pick policy returned empty name")
	}
	if got := Pick(th, "nonsense-kind", 0); !strings.HasPrefix(got.Name, "item-") {
		t.Errorf("expected fallback name for unknown kind, got %q", got.Name)
	}
}

func TestGetUnknownErrors(t *testing.T) {
	if _, err := Get("notreal"); err == nil {
		t.Error("expected error for unknown theme")
	}
}
