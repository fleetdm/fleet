package perfconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample(id string) Config {
	return Config{
		ID:             id,
		Name:           "load test",
		ServerURL:      "https://localhost:8080",
		EnrollSecret:   "secret",
		OSCounts:       map[string]uint32{"macos_14.1.2": 10, "ubuntu_22.04": 5},
		MDMEnabled:     true,
		MDMProb:        0.5,
		StartPeriod:    "10s",
		QueryInterval:  "10s",
		ConfigInterval: "1m",
	}
}

func TestListMissingFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	got, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("List returned nil slice, want empty non-nil")
	}
	if len(got) != 0 {
		t.Fatalf("List = %d entries, want 0", len(got))
	}
}

func TestSaveNewStampsTimestamps(t *testing.T) {
	dir := t.TempDir()
	saved, err := Save(dir, sample("a"), 1000)
	if err != nil {
		t.Fatal(err)
	}
	if saved.CreatedAtMS != 1000 || saved.UpdatedAtMS != 1000 {
		t.Errorf("new save timestamps = created %d updated %d, want 1000/1000", saved.CreatedAtMS, saved.UpdatedAtMS)
	}

	list, _ := List(dir)
	if len(list) != 1 || list[0].ID != "a" {
		t.Fatalf("after save, list = %+v", list)
	}
}

func TestSaveNewKeepsProvidedCreatedAt(t *testing.T) {
	dir := t.TempDir()
	c := sample("a")
	c.CreatedAtMS = 42 // caller supplied a non-zero created stamp
	saved, err := Save(dir, c, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if saved.CreatedAtMS != 42 {
		t.Errorf("CreatedAtMS = %d, want preserved 42", saved.CreatedAtMS)
	}
	if saved.UpdatedAtMS != 1000 {
		t.Errorf("UpdatedAtMS = %d, want 1000", saved.UpdatedAtMS)
	}
}

func TestSaveExistingUpsert(t *testing.T) {
	dir := t.TempDir()
	if _, err := Save(dir, sample("a"), 1000); err != nil {
		t.Fatal(err)
	}

	upd := sample("a")
	upd.Name = "renamed"
	upd.CreatedAtMS = 999999 // should be ignored — original is preserved
	saved, err := Save(dir, upd, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if saved.CreatedAtMS != 1000 {
		t.Errorf("CreatedAtMS = %d, want original 1000 preserved on update", saved.CreatedAtMS)
	}
	if saved.UpdatedAtMS != 2000 {
		t.Errorf("UpdatedAtMS = %d, want 2000", saved.UpdatedAtMS)
	}

	list, _ := List(dir)
	if len(list) != 1 {
		t.Fatalf("upsert created a duplicate: %d entries", len(list))
	}
	if list[0].Name != "renamed" {
		t.Errorf("Name = %q, want renamed", list[0].Name)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	mustSave(t, dir, "a")
	mustSave(t, dir, "b")

	if err := Delete(dir, "a"); err != nil {
		t.Fatal(err)
	}
	list, _ := List(dir)
	if len(list) != 1 || list[0].ID != "b" {
		t.Fatalf("after delete, list = %+v", list)
	}

	// Deleting a missing id is a no-op, not an error.
	if err := Delete(dir, "nope"); err != nil {
		t.Errorf("Delete(missing) = %v, want nil", err)
	}
}

// The on-disk format must be {"configs": []} for an empty list, never null —
// the Rust app emits [] and we keep the file byte-compatible.
func TestEmptyListSerializesAsArray(t *testing.T) {
	dir := t.TempDir()
	mustSave(t, dir, "a")
	if err := Delete(dir, "a"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "null") {
		t.Errorf("empty configs serialized with null:\n%s", raw)
	}
	if !strings.Contains(string(raw), "\"configs\": []") {
		t.Errorf("expected \"configs\": [], got:\n%s", raw)
	}
}

func TestJSONKeysAndSortedOSCounts(t *testing.T) {
	dir := t.TempDir()
	mustSave(t, dir, "a")
	raw, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"\"server_url\"", "\"enroll_secret\"", "\"os_counts\"", "\"mdm_enabled\"",
		"\"mdm_prob\"", "\"mdm_scep_challenge\"", "\"created_at_ms\"", "\"updated_at_ms\"",
	} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("serialized config missing %s:\n%s", key, raw)
		}
	}

	// os_counts keys must be alphabetically sorted (BTreeMap parity).
	var f configsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if i, j := strings.Index(body, "macos_14.1.2"), strings.Index(body, "ubuntu_22.04"); i < 0 || j < 0 || i > j {
		t.Errorf("os_counts keys not sorted (macos before ubuntu): i=%d j=%d", i, j)
	}
}

func mustSave(t *testing.T, dir, id string) {
	t.Helper()
	if _, err := Save(dir, sample(id), 1000); err != nil {
		t.Fatalf("save %s: %v", id, err)
	}
}
