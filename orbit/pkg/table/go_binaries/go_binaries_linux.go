//go:build linux

package go_binaries

import (
	"bufio"
	"context"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// GenerateGoBinaries is called to return the results for the go_binaries table at query time.
func GenerateGoBinaries(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	homeDirs, err := linuxHomeDirs()
	if err != nil {
		log.Debug().Err(err).Msg("go_binaries: failed to list home directories")
		return nil, nil
	}
	return generateForDirs(homeDirs), nil
}

// linuxHomeDirs returns home directories for real users by parsing /etc/passwd.
func linuxHomeDirs() ([]string, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parsePasswdHomeDirs(f)
}

// parsePasswdHomeDirs parses /etc/passwd-formatted content and returns home
// directories for real (non-system) user accounts.
func parsePasswdHomeDirs(r io.Reader) ([]string, error) {
	seen := make(map[string]bool)
	var dirs []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), ":", 7)
		if len(fields) < 7 {
			continue
		}
		// fields: username:password:uid:gid:gecos:home:shell
		uid, _ := strconv.Atoi(fields[2])
		home := fields[5]
		shell := fields[6]
		// Skip system accounts, accounts with nologin/false shells,
		// and accounts with no real home directory.
		if uid < 500 || home == "" || home == "/" || home == "/dev/null" {
			continue
		}
		if strings.HasSuffix(shell, "/nologin") || strings.HasSuffix(shell, "/false") {
			continue
		}
		if !seen[home] {
			seen[home] = true
			dirs = append(dirs, home)
		}
	}
	return dirs, scanner.Err()
}
