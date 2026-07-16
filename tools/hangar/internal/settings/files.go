package settings

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
)

// yamlExts gate the generic text read/write commands; openExts gate which
// files open_path will hand to `open`. Both reject executables.
var (
	yamlExts = []string{"yml", "yaml"}
	openExts = []string{"yml", "yaml", "log", "json", "txt", "md", "sql", "gz"}
)

// ReadTextFile reads a .yml/.yaml file under $HOME. The extension allowlist
// and the under-$HOME guard make this safe to expose to the webview.
func ReadTextFile(path string) (string, error) {
	resolved := paths.Expand(path)
	if err := paths.UnderHome(resolved); err != nil {
		return "", err
	}
	if !paths.HasExt(resolved, yamlExts...) {
		return "", errors.New("only .yml/.yaml files supported")
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteTextFile writes a .yml/.yaml file under $HOME.
func WriteTextFile(path, contents string) error {
	resolved := paths.Expand(path)
	if err := paths.UnderHome(resolved); err != nil {
		return err
	}
	if !paths.HasExt(resolved, yamlExts...) {
		return errors.New("only .yml/.yaml files supported")
	}
	return os.WriteFile(resolved, []byte(contents), 0o644)
}

// OpenPath opens a directory, or a file with an allowed plain-text/config
// extension, in the system file manager / default app. Anything executable
// is rejected. reveal=true selects the item in Finder (open -R).
func OpenPath(path string, reveal bool) error {
	resolved := paths.Expand(path)
	if err := paths.UnderHome(resolved); err != nil {
		return err
	}
	info, err := os.Stat(resolved)
	isDir := err == nil && info.IsDir()
	if !isDir && !paths.HasExt(resolved, openExts...) {
		return errors.New("unsupported file type for open")
	}
	args := make([]string, 0, 2)
	if reveal {
		args = append(args, "-R")
	}
	args = append(args, resolved)
	return exec.Command("open", args...).Run()
}

// OpenURL opens an http(s) URL in the default browser. Kept narrow so the
// webview can't smuggle a file:// or custom-handler URL through here.
func OpenURL(url string) error {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return errors.New("only http(s) URLs are allowed")
	}
	return exec.Command("open", url).Run()
}
