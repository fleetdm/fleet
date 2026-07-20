// Package mdmassets runs Fleet's in-repo tools/mdm/assets exporter and persists
// saved export configurations to <config-dir>/mdm-assets-configs.json (same
// shape/pattern as the perfconfig package). Each config remembers the MySQL
// connection (the tool's four consts), the encryption key, the output dir, and
// an optional single-asset filter — so an export is one button press.
//
// Like perfconfig, this stores dev-only local credentials as plain text: the
// same security boundary as ~/.fleet/config and the rest of the Hangar config.
//
// All persistence functions take an explicit directory so they're hermetically
// testable; the service layer resolves the real config dir via the paths pkg.
package mdmassets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

const fileName = "mdm-assets-configs.json"

// toolPkg is the in-repo exporter, built/run from the primary Fleet repo.
const toolPkg = "./tools/mdm/assets"

// Config is one saved MDM-assets export configuration.
type Config struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// MySQL connection — defaults mirror the tool's consts (fleet / insecure /
	// localhost:3306 / fleet).
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBAddress  string `json:"db_address"`
	DBName     string `json:"db_name"`
	// Key is the server private key used to decrypt the assets (export's -key,
	// required).
	Key string `json:"key"`
	// Dir is the export output directory; empty means the Fleet repo root.
	Dir string `json:"dir"`
	// AssetName optionally limits the export to a single asset (export's -name);
	// empty exports the full set.
	AssetName   string `json:"asset_name"`
	CreatedAtMS uint64 `json:"created_at_ms"`
	UpdatedAtMS uint64 `json:"updated_at_ms"`
}

// AssetFile is one file the exporter wrote.
type AssetFile struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	ModTimeMS int64  `json:"mod_time_ms"`
}

// ExportResult is a finished export run: captured output plus the files it
// produced. ExitCode is nil if the process was killed/timed out.
type ExportResult struct {
	ExitCode *int        `json:"exit_code"`
	Stdout   string      `json:"stdout"`
	Stderr   string      `json:"stderr"`
	Files    []AssetFile `json:"files"`
}

// ExportArgs builds the exporter's args for a config against a resolved output
// dir. Pure; unit-tested.
func ExportArgs(cfg Config, absDir string) []string {
	args := []string{
		"export",
		"-key", cfg.Key,
		"-dir", absDir,
		"-db-user", cfg.DBUser,
		"-db-password", cfg.DBPassword,
		"-db-address", cfg.DBAddress,
		"-db-name", cfg.DBName,
	}
	if strings.TrimSpace(cfg.AssetName) != "" {
		args = append(args, "-name", strings.TrimSpace(cfg.AssetName))
	}
	return args
}

// writtenRe matches the exporter's `wrote <name> in <path>` log lines (which
// carry a leading `log` timestamp). The path runs to end-of-line so dirs with
// spaces are captured whole.
var writtenRe = regexp.MustCompile(`(?m)wrote \S+ in (.+?)\s*$`)

// ParseWrittenPaths extracts the file paths the exporter reported writing.
// Pure; unit-tested.
func ParseWrittenPaths(output string) []string {
	matches := writtenRe.FindAllStringSubmatch(output, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if p := strings.TrimSpace(m[1]); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// StatFiles turns a set of paths into AssetFiles, skipping any that don't exist.
func StatFiles(paths []string) []AssetFile {
	files := make([]AssetFile, 0, len(paths))
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		files = append(files, AssetFile{
			Name:      filepath.Base(p),
			Path:      p,
			Size:      st.Size(),
			ModTimeMS: st.ModTime().UnixMilli(),
		})
	}
	return files
}

// Export runs `go run ./tools/mdm/assets export …` from repoPath (so the Fleet
// module resolves) writing to absDir, and returns the captured output + the
// files produced. The tool's own failures surface via ExportResult (ExitCode /
// Stderr), not the returned error — that's reserved for setup problems — so the
// UI can show partial output.
func Export(ctx context.Context, repoPath string, cfg Config, absDir string) (ExportResult, error) {
	if strings.TrimSpace(repoPath) == "" {
		return ExportResult{}, errors.New("no primary repo configured — set server 1's repo first")
	}
	args := append([]string{"run", toolPkg}, ExportArgs(cfg, absDir)...)
	cmd := shellpath.CommandContext(ctx, "go", args...)
	cmd.Dir = repoPath
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	res := ExportResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if runErr == nil {
		zero := 0
		res.ExitCode = &zero
	} else {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) && ee.ProcessState.Exited() {
			code := ee.ProcessState.ExitCode()
			res.ExitCode = &code
		}
		// else (context timeout / signal): leave ExitCode nil.
	}
	// Parse regardless of exit — some files may be written before a later error.
	res.Files = StatFiles(ParseWrittenPaths(res.Stdout + "\n" + res.Stderr))
	return res, nil
}

// --- persistence (mirrors internal/perfconfig) ---

type configsFile struct {
	Configs []Config `json:"configs"`
}

func path(dir string) string { return filepath.Join(dir, fileName) }

// List returns all saved configs, or an empty slice if the file is absent.
func List(dir string) ([]Config, error) { return readAll(dir) }

func readAll(dir string) ([]Config, error) {
	b, err := os.ReadFile(path(dir))
	if errors.Is(err, fs.ErrNotExist) {
		return []Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", fileName, err)
	}
	var f configsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", fileName, err)
	}
	if f.Configs == nil {
		f.Configs = []Config{}
	}
	return f.Configs, nil
}

func writeAll(dir string, configs []Config) error {
	if configs == nil {
		configs = []Config{}
	}
	b, err := json.MarshalIndent(configsFile{Configs: configs}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path(dir), b, 0o644)
}

// Save upserts cfg by ID (preserving the original CreatedAtMS), stamping
// nowMS into UpdatedAtMS (and CreatedAtMS for a new record).
func Save(dir string, cfg Config, nowMS uint64) (Config, error) {
	all, err := readAll(dir)
	if err != nil {
		return Config{}, err
	}
	for i := range all {
		if all[i].ID == cfg.ID {
			cfg.CreatedAtMS = all[i].CreatedAtMS
			cfg.UpdatedAtMS = nowMS
			all[i] = cfg
			if err := writeAll(dir, all); err != nil {
				return Config{}, err
			}
			return cfg, nil
		}
	}
	if cfg.CreatedAtMS == 0 {
		cfg.CreatedAtMS = nowMS
	}
	cfg.UpdatedAtMS = nowMS
	all = append(all, cfg)
	if err := writeAll(dir, all); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Delete removes the config with the given ID (a no-op if absent).
func Delete(dir, id string) error {
	all, err := readAll(dir)
	if err != nil {
		return err
	}
	out := make([]Config, 0, len(all))
	for _, c := range all {
		if c.ID != id {
			out = append(out, c)
		}
	}
	return writeAll(dir, out)
}
