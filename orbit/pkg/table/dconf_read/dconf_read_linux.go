//go:build linux
// +build linux

package dconf_read

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// NixOS has no /usr/bin: sudo is a setuid wrapper in /run/wrappers/bin and system
// packages are linked into /run/current-system/sw/bin.
var (
	sudoPaths  = []string{"/usr/bin/sudo", "/run/wrappers/bin/sudo"}
	dconfPaths = []string{"/usr/bin/dconf", "/run/current-system/sw/bin/dconf"}
)

func firstExisting(paths []string) string {
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("username"), // required
		table.TextColumn("key"),      // required
		table.TextColumn("value"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	getFirstConstraint := func(columnName string) string {
		if constraints, ok := queryContext.Constraints[columnName]; ok {
			for _, constraint := range constraints.Constraints {
				if constraint.Operator == table.OperatorEquals {
					return constraint.Expression
				}
			}
		}
		return ""
	}

	username := getFirstConstraint("username")
	if username == "" {
		return nil, errors.New("missing username")
	}

	key := getFirstConstraint("key")
	if key == "" {
		return nil, errors.New("missing key")
	}

	sudoPath := firstExisting(sudoPaths)
	if sudoPath == "" {
		return nil, fmt.Errorf("sudo not found in %v", sudoPaths)
	}
	dconfCandidates := slices.Clone(dconfPaths)
	if !strings.ContainsRune(username, '/') {
		dconfCandidates = append(dconfCandidates, filepath.Join("/etc/profiles/per-user", username, "bin", "dconf"))
	}
	dconfPath := firstExisting(dconfCandidates)
	if dconfPath == "" {
		return nil, fmt.Errorf("dconf not found in %v", dconfCandidates)
	}

	cmd := exec.Command(sudoPath, "-u", username, dconfPath, "read", key)
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	log.Debug().Str("cmd", cmd.String()).Msg("running")
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("dconf read failed: %w: %s", err, stderr.String())
	}

	return []map[string]string{{
		"username": username,
		"key":      key,
		"value":    stdout.String(),
	}}, nil
}
