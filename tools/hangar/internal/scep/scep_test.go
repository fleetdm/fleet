package scep

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

func TestProcIDAndLogChannel(t *testing.T) {
	if got := ProcID("scep1"); got != "scep:scep1" {
		t.Errorf("ProcID = %q, want scep:scep1", got)
	}
	if got := LogChannel("scep1"); got != "scep-scep1" {
		t.Errorf("LogChannel = %q, want scep-scep1", got)
	}
}

func TestServeArgs(t *testing.T) {
	p := settings.ScepProfile{Port: 2017, Challenge: "secret", AllowRenew: 0, Debug: true, ExtraFlags: "-sign-server-attrs"}
	got := ServeArgs("/d/win", p)
	want := []string{"-depot", "/d/win", "-port", "2017", "-allowrenew", "0", "-challenge", "secret", "-debug", "-sign-server-attrs"}
	if !equal(got, want) {
		t.Errorf("ServeArgs = %v, want %v", got, want)
	}

	// No challenge, no debug, no extra flags.
	got = ServeArgs("/d/x", settings.ScepProfile{Port: 2016, AllowRenew: 14})
	want = []string{"-depot", "/d/x", "-port", "2016", "-allowrenew", "14"}
	if !equal(got, want) {
		t.Errorf("ServeArgs (minimal) = %v, want %v", got, want)
	}
}

func TestInitCAArgs(t *testing.T) {
	got := InitCAArgs("/d/new", InitCAParams{
		CommonName: "Fleet Windows SCEP CA", Organization: "Fleet Device Management Inc.",
		OrganizationalUnit: "Windows QA", Country: "US", KeySize: 2048, Years: 10,
	})
	want := []string{
		"ca", "-init", "-depot", "/d/new",
		"-common_name", "Fleet Windows SCEP CA",
		"-organization", "Fleet Device Management Inc.",
		"-organizational_unit", "Windows QA",
		"-country", "US", "-keySize", "2048", "-years", "10",
	}
	if !equal(got, want) {
		t.Errorf("InitCAArgs = %v, want %v", got, want)
	}

	// Zero key size/years fall back to binary defaults; key-password appended.
	got = InitCAArgs("/d/new", InitCAParams{CommonName: "X", KeyPassword: "pw"})
	if !contains(got, "-keySize", "4096") || !contains(got, "-years", "10") {
		t.Errorf("expected default keySize 4096 / years 10, got %v", got)
	}
	if !contains(got, "-key-password", "pw") {
		t.Errorf("expected -key-password pw, got %v", got)
	}
}

func TestParseDepotNotInitialized(t *testing.T) {
	info := ParseDepot(t.TempDir()) // empty dir, no ca.pem
	if info.Exists {
		t.Error("empty depot should report Exists=false")
	}
	if info.Error != "" {
		t.Errorf("missing ca.pem should not be an error, got %q", info.Error)
	}
}

func TestParseDepotBadPEM(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ca.pem"), []byte("not a pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	info := ParseDepot(dir)
	if info.Exists || info.Error == "" {
		t.Errorf("bad PEM should set Error and Exists=false, got %+v", info)
	}
}

func TestParseDepotValidCert(t *testing.T) {
	dir := t.TempDir()
	writeTestCA(t, dir, "MICROMDM SCEP CA", "scep-ca")

	info := ParseDepot(dir)
	if !info.Exists {
		t.Fatalf("expected Exists=true, got %+v", info)
	}
	if len(info.Thumbprint) != 40 {
		t.Errorf("thumbprint should be 40 hex chars, got %q (%d)", info.Thumbprint, len(info.Thumbprint))
	}
	if info.Thumbprint != up(info.Thumbprint) {
		t.Errorf("thumbprint should be uppercase, got %q", info.Thumbprint)
	}
	if !containsSub(info.IssuerDN, "CN=MICROMDM SCEP CA") {
		t.Errorf("issuer DN = %q, want it to contain the CN", info.IssuerDN)
	}
	if info.NotAfter == "" {
		t.Error("NotAfter should be populated")
	}
}

func TestStatBinary(t *testing.T) {
	dir := t.TempDir()
	if got := StatBinary(dir); got.Exists {
		t.Error("no binary yet should report Exists=false")
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "scepserver"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := StatBinary(dir)
	if !got.Exists || got.BuiltAt == "" {
		t.Errorf("expected Exists=true with BuiltAt, got %+v", got)
	}
}

// --- test helpers ---

func writeTestCA(t *testing.T, dir, cn, org string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn, Organization: []string{org}},
		Issuer:                pkix.Name{CommonName: cn, Organization: []string{org}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(filepath.Join(dir, "ca.pem"), pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// contains reports whether flag immediately followed by val appears in args.
func contains(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}

func containsSub(s, sub string) bool { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func up(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
