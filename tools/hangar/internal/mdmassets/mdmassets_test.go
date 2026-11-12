package mdmassets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExportArgs(t *testing.T) {
	cfg := Config{
		Key: "k", DBUser: "fleet", DBPassword: "insecure",
		DBAddress: "localhost:3306", DBName: "fleet",
	}
	got := ExportArgs(cfg, "/out")
	want := []string{
		"export", "-key", "k", "-dir", "/out",
		"-db-user", "fleet", "-db-password", "insecure",
		"-db-address", "localhost:3306", "-db-name", "fleet",
	}
	if !equal(got, want) {
		t.Errorf("ExportArgs = %v, want %v", got, want)
	}

	// Single-asset filter appends -name.
	cfg.AssetName = "apns_cert"
	got = ExportArgs(cfg, "/out")
	if !contains(got, "-name", "apns_cert") {
		t.Errorf("expected -name apns_cert, got %v", got)
	}

	// Blank asset name is not passed.
	cfg.AssetName = "  "
	if contains2(ExportArgs(cfg, "/out"), "-name") {
		t.Errorf("blank asset name should not add -name")
	}
}

func TestParseWrittenPaths(t *testing.T) {
	out := `2026/07/17 00:43:46 wrote ca_cert in /Users/a/fleet/ca_cert.crt
2026/07/17 00:43:46 wrote scep_challenge in /Users/a/fleet/scep_challenge
some other line
2026/07/17 00:43:46 wrote apns_key in /Users/a/my fleet/apns_key.key`
	got := ParseWrittenPaths(out)
	want := []string{
		"/Users/a/fleet/ca_cert.crt",
		"/Users/a/fleet/scep_challenge",
		"/Users/a/my fleet/apns_key.key", // path with a space captured whole
	}
	if !equal(got, want) {
		t.Errorf("ParseWrittenPaths = %v, want %v", got, want)
	}

	if len(ParseWrittenPaths("nothing here")) != 0 {
		t.Error("expected no paths from output with no wrote lines")
	}
}

func TestSaveListDeleteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got, _ := List(dir); len(got) != 0 {
		t.Errorf("empty dir should list 0 configs, got %d", len(got))
	}

	saved, err := Save(dir, Config{ID: "a", Name: "local", DBName: "fleet"}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if saved.CreatedAtMS != 1000 || saved.UpdatedAtMS != 1000 {
		t.Errorf("timestamps not stamped: %+v", saved)
	}

	// Update preserves CreatedAtMS, bumps UpdatedAtMS.
	saved2, err := Save(dir, Config{ID: "a", Name: "local2", DBName: "fleet"}, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if saved2.CreatedAtMS != 1000 || saved2.UpdatedAtMS != 2000 {
		t.Errorf("update should preserve created/bump updated: %+v", saved2)
	}
	if list, _ := List(dir); len(list) != 1 || list[0].Name != "local2" {
		t.Errorf("expected 1 config named local2, got %+v", list)
	}

	if err := Delete(dir, "a"); err != nil {
		t.Fatal(err)
	}
	if list, _ := List(dir); len(list) != 0 {
		t.Errorf("expected 0 configs after delete, got %d", len(list))
	}
}

func TestStatFilesSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "ca_cert.crt")
	if err := writeFile(t, real, "hello"); err != nil {
		t.Fatal(err)
	}
	files := StatFiles([]string{real, filepath.Join(dir, "missing")})
	if len(files) != 1 {
		t.Fatalf("expected 1 file (missing skipped), got %d", len(files))
	}
	if files[0].Name != "ca_cert.crt" || files[0].Size != 5 || files[0].ModTimeMS == 0 {
		t.Errorf("unexpected file info: %+v", files[0])
	}
}

// --- helpers ---

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

func contains(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}

func contains2(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func writeFile(t *testing.T, path, contents string) error {
	t.Helper()
	return os.WriteFile(path, []byte(contents), 0o600)
}
