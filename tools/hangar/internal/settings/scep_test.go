package settings

import (
	"path/filepath"
	"testing"
)

func TestNextScepProfileDefaults(t *testing.T) {
	p := NextScepProfile(nil)
	if p.ID != "scep1" {
		t.Errorf("id = %q, want scep1", p.ID)
	}
	if p.Port != defaultScepStartPort {
		t.Errorf("port = %d, want %d", p.Port, defaultScepStartPort)
	}
	if p.Challenge != "secret" {
		t.Errorf("challenge = %q, want secret", p.Challenge)
	}
	if p.AllowRenew != 0 {
		t.Errorf("allow_renew = %d, want 0", p.AllowRenew)
	}
	if !p.Debug {
		t.Error("debug should default true")
	}
}

func TestNextScepProfileUniqueIDAndPort(t *testing.T) {
	existing := []ScepProfile{
		{ID: "scep1", Port: 2016},
		{ID: "scep2", Port: 2017},
	}
	p := NextScepProfile(existing)
	if p.ID != "scep3" {
		t.Errorf("id = %q, want scep3", p.ID)
	}
	if p.Port != 2018 {
		t.Errorf("port = %d, want 2018 (first free)", p.Port)
	}
}

func TestNextScepProfileSkipsUsedID(t *testing.T) {
	// A middle profile was removed and re-added; ids must never be reused.
	existing := []ScepProfile{
		{ID: "scep1", Port: 2016},
		{ID: "scep3", Port: 2018},
	}
	p := NextScepProfile(existing)
	if p.ID != "scep4" {
		t.Errorf("id = %q, want scep4 (scep3 is taken so len-derived scep3 is skipped)", p.ID)
	}
	if p.Port != 2017 {
		t.Errorf("port = %d, want 2017 (lowest free at/above 2016)", p.Port)
	}
}

func TestResolveDepotPath(t *testing.T) {
	dir := "/data/scep-depots"

	managed := ResolveDepotPath(dir, ScepProfile{ID: "scep1"})
	if want := filepath.Join(dir, "scep1"); managed != want {
		t.Errorf("managed depot = %q, want %q", managed, want)
	}

	explicit := ResolveDepotPath(dir, ScepProfile{ID: "scep1", DepotPath: "/custom/depot"})
	if explicit != "/custom/depot" {
		t.Errorf("explicit depot = %q, want /custom/depot", explicit)
	}
}

func TestMigrateNormalizesNilScepProfiles(t *testing.T) {
	s := Settings{} // no Servers, no ScepProfiles — like a pre-SCEP file
	migrate(&s)
	if s.ScepProfiles == nil {
		t.Error("migrate should normalize nil ScepProfiles to an empty slice")
	}
	if len(s.ScepProfiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(s.ScepProfiles))
	}
}

func TestScepProfilesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := Default()
	depots := "/data/scep-depots"
	in.ScepDepotsDir = &depots
	in.ScepProfiles = []ScepProfile{
		{ID: "scep1", Name: "windows", DepotPath: "/d/win", Port: 2017, Challenge: "secret", AllowRenew: 0, Debug: true, ExtraFlags: "-sign-server-attrs"},
	}
	if err := Save(dir, in); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.ScepDepotsDir == nil || *got.ScepDepotsDir != depots {
		t.Errorf("scep_depots_dir round-trip failed: %v", got.ScepDepotsDir)
	}
	if len(got.ScepProfiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(got.ScepProfiles))
	}
	p := got.ScepProfiles[0]
	if p.Name != "windows" || p.Port != 2017 || p.ExtraFlags != "-sign-server-attrs" || !p.Debug {
		t.Errorf("profile round-trip mismatch: %+v", p)
	}
}
