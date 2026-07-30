package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYml(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ngrok.yml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseNgrokV2Token(t *testing.T) {
	p := writeYml(t, `authtoken: abc123
tunnels:
  web:
    proto: http
    addr: localhost:8000
  fleet:
    proto: http
    addr: 8080
`)
	got := ParseNgrokYml(p)
	if !got.Valid {
		t.Fatalf("expected valid, got %+v", got)
	}
	if !got.HasAuthtoken {
		t.Error("v2 top-level authtoken not detected")
	}
	if len(got.Tunnels) != 2 {
		t.Fatalf("got %d tunnels, want 2", len(got.Tunnels))
	}
	// Sorted by name: fleet before web.
	if got.Tunnels[0].Name != "fleet" || got.Tunnels[1].Name != "web" {
		t.Errorf("tunnels not sorted by name: %+v", got.Tunnels)
	}
	// addr from a bare number and from a host:port string.
	if got.Tunnels[0].Addr != "8080" {
		t.Errorf("numeric addr = %q, want 8080", got.Tunnels[0].Addr)
	}
	if got.Tunnels[1].Addr != "localhost:8000" {
		t.Errorf("string addr = %q, want localhost:8000", got.Tunnels[1].Addr)
	}
}

func TestParseNgrokV3Token(t *testing.T) {
	p := writeYml(t, `version: "3"
agent:
  authtoken: xyz789
`)
	got := ParseNgrokYml(p)
	if !got.Valid || !got.HasAuthtoken {
		t.Errorf("v3 agent.authtoken not detected: %+v", got)
	}
}

func TestParseNgrokEmptyToken(t *testing.T) {
	p := writeYml(t, "authtoken: \"   \"\nagent:\n  authtoken: \"\"\n")
	got := ParseNgrokYml(p)
	if got.HasAuthtoken {
		t.Error("blank tokens should not count as present")
	}
}

func TestParseNgrokMissingFile(t *testing.T) {
	got := ParseNgrokYml(filepath.Join(t.TempDir(), "nope.yml"))
	if got.Valid {
		t.Error("missing file should be invalid")
	}
	if got.Error == nil || *got.Error != "file not found" {
		t.Errorf("error = %v, want 'file not found'", got.Error)
	}
}

func TestParseNgrokBadYaml(t *testing.T) {
	p := writeYml(t, "this: : : not valid yaml\n  - broken")
	got := ParseNgrokYml(p)
	if got.Valid || got.Error == nil {
		t.Errorf("malformed yaml should be invalid with error: %+v", got)
	}
}
