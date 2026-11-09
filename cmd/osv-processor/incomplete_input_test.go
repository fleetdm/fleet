package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIncompleteOSVInputPreservesArtifacts(t *testing.T) {
	for _, platform := range []struct {
		name      string
		ecosystem string
		version   string
		run       func(Config) error
	}{
		{"ubuntu", "Ubuntu:24.04:LTS", "24.04", run},
		{"rhel", "Red Hat:enterprise_linux:9::baseos", "9", runRHEL},
	} {
		for _, inputCase := range []string{
			"empty", "non-JSON only", "malformed only", "valid before malformed", "malformed before valid",
			"null", "missing ID",
		} {
			t.Run(platform.name+"/"+inputCase, func(t *testing.T) {
				input, output := t.TempDir(), t.TempDir()
				cfg := Config{
					Platform: platform.name, InputDir: input, OutputDir: output, Versions: platform.version,
					DateStr: "2026-11-03", YesterdayStr: "2026-11-02",
					GeneratedTimestamp: "2026-11-03T00:00:00Z",
					RunTime:            time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
				}
				goodPath := filepath.Join(input, "CVE-2026-12345.json")
				entry := OSVData{
					ID: "CVE-2026-12345", SchemaVersion: "1.7.5", Upstream: []string{"CVE-2026-12345"},
					Affected: []Affected{{Package: Package{Ecosystem: platform.ecosystem, Name: "test-package"}}},
				}
				data, err := json.Marshal(entry)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(goodPath, data, 0o600))
				require.NoError(t, platform.run(cfg))
				for _, name := range []string{
					"osv-" + platform.name + "-other-" + cfg.DateStr + ".json.gz",
					"osv-" + platform.name + "-" + strings.ReplaceAll(platform.version, ".", "") + "-2026-11-01.json.gz",
					"osv-other-9-" + cfg.DateStr + ".json.gz",
				} {
					require.NoError(t, os.WriteFile(filepath.Join(output, name), []byte("earlier run"), 0o600))
				}
				readOutput := func() map[string][]byte {
					t.Helper()
					files, err := os.ReadDir(output)
					require.NoError(t, err)
					result := make(map[string][]byte, len(files))
					for _, file := range files {
						require.False(t, file.IsDir())
						data, err := os.ReadFile(filepath.Join(output, file.Name()))
						require.NoError(t, err)
						result[file.Name()] = data
					}
					return result
				}
				before := readOutput()
				switch inputCase {
				case "empty", "non-JSON only", "malformed only":
					require.NoError(t, os.Remove(goodPath))
					if inputCase == "non-JSON only" {
						require.NoError(t, os.WriteFile(filepath.Join(input, "README.txt"), []byte("incomplete download"), 0o600))
					}
					if inputCase == "malformed only" {
						require.NoError(t, os.WriteFile(goodPath, []byte("{"), 0o600))
					}
				case "valid before malformed":
					require.NoError(t, os.WriteFile(filepath.Join(input, "zz-broken.json"), []byte("{"), 0o600))
				case "malformed before valid":
					require.NoError(t, os.WriteFile(filepath.Join(input, "000-broken.json"), []byte("{"), 0o600))
				case "null":
					require.NoError(t, os.WriteFile(goodPath, []byte("null"), 0o600))
				case "missing ID":
					require.NoError(t, os.WriteFile(goodPath, []byte("{}"), 0o600))
				}
				require.Error(t, platform.run(cfg))
				require.Equal(t, before, readOutput(), "an incomplete input must neither delete nor rewrite existing artifacts")
			})
		}
	}
}
