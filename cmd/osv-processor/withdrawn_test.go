package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithdrawnAdvisories(t *testing.T) {
	for _, tc := range []struct {
		platform  string
		ecosystem string
		version   string
		fixed     string
		run       func(Config) error
	}{
		{"ubuntu", "Ubuntu:24.04:LTS", "2404", "1.2.3", run},
		{"rhel", "Red Hat:enterprise_linux:9::baseos", "9", "1.2.3", runRHEL},
		{"android", "Android", "16", "16:2026-10-01", runAndroid},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			input, output := t.TempDir(), t.TempDir()
			cfg := Config{
				Platform: tc.platform, InputDir: input, OutputDir: output,
				DateStr: "2026-11-03", YesterdayStr: "2026-11-02",
				GeneratedTimestamp: "2026-11-03T00:00:00Z",
				RunTime:            time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
			}
			ids := []string{"CVE-2026-12345", "CVE-2026-23456"}
			write := func(id, withdrawn string) {
				t.Helper()
				entry := map[string]any{
					"id": id, "schema_version": "1.7.5",
					"published": "2026-10-01T00:00:00Z", "modified": "2026-10-01T00:00:00Z",
					"upstream": []string{id}, "aliases": []string{id},
					"affected": []any{map[string]any{
						"package": map[string]string{"name": "test-package", "ecosystem": tc.ecosystem},
						"ranges": []any{map[string]any{
							"type":   "ECOSYSTEM",
							"events": []any{map[string]string{"introduced": "0"}, map[string]string{"fixed": tc.fixed}},
						}},
						"versions": []string{"1.0"},
					}},
				}
				if withdrawn != "" {
					entry["withdrawn"] = withdrawn
				}
				data, err := json.Marshal(entry)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(input, id+".json"), data, 0o600))
			}
			for _, id := range ids {
				write(id, "")
			}
			// A withdrawn duplicate must not shadow the active record in RHEL's deduplication.
			duplicate, err := os.ReadFile(filepath.Join(input, ids[1]+".json"))
			require.NoError(t, err)
			var entry map[string]any
			require.NoError(t, json.Unmarshal(duplicate, &entry))
			entry["withdrawn"] = "2026-11-01T00:00:00Z"
			duplicate, err = json.Marshal(entry)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(input, "000-withdrawn-duplicate.json"), duplicate, 0o600))
			files := []string{"osv-" + tc.platform + "-" + tc.version + "-" + cfg.DateStr + ".json.gz"}
			if tc.platform == "ubuntu" {
				changed := filepath.Join(t.TempDir(), "changed.txt")
				require.NoError(t, os.WriteFile(changed, []byte("osv/cve/"+ids[0]+".json\nosv/cve/"+ids[1]+".json\n"), 0o600))
				cfg.ChangedFilesToday, cfg.ChangedFilesYesterday = changed, changed
				for _, date := range []string{cfg.DateStr, cfg.YesterdayStr} {
					files = append(files, "osv-ubuntu-2404-delta-"+date+".json.gz")
				}
			}
			check := func(want []string) {
				t.Helper()
				require.NoError(t, tc.run(cfg))
				for _, name := range files {
					var got []string
					if tc.platform == "android" {
						artifact, err := readAndroidArtifact(filepath.Join(output, name))
						require.NoError(t, err)
						require.Equal(t, len(want), artifact.TotalCVEs)
						for _, vuln := range artifact.Vulnerabilities {
							require.Equal(t, "2026-10-01", vuln.FixedSPL)
							require.Empty(t, vuln.Severity)
							got = append(got, vuln.CVE)
						}
					} else {
						artifact, err := readArtifact(filepath.Join(output, name))
						require.NoError(t, err)
						require.Equal(t, len(want), artifact.TotalCVEs)
						require.Equal(t, min(len(want), 1), artifact.TotalPackages)
						for _, vulns := range artifact.Vulnerabilities {
							for _, vuln := range vulns {
								require.Equal(t, ProcessedVuln{
									CVE: vuln.CVE, Published: "2026-10-01T00:00:00Z", Modified: "2026-10-01T00:00:00Z",
									Introduced: "0", Fixed: tc.fixed, Versions: []string{"1.0"},
								}, vuln)
								got = append(got, vuln.CVE)
							}
						}
					}
					require.ElementsMatch(t, want, got, name)
				}
			}
			check(ids)
			// Only withdrawn changes: retained affected data and modified time must not keep the CVE alive.
			write(ids[0], "2026-11-01T00:00:00Z")
			check(ids[1:])
		})
	}
}
