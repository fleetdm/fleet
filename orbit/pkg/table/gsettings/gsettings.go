//go:build !windows
// +build !windows

package gsettings

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

const gsettingsPath = "/usr/bin/gsettings"

const allowedCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-."

type gsettingsExecer func(ctx context.Context, username string, buf *bytes.Buffer) error

type GsettingsValues struct {
	logger   log.Logger
	getBytes gsettingsExecer
}

// Settings returns a table plugin for querying setting values from the
// gsettings command.
func Settings(logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("schema"),
		table.TextColumn("key"),
		table.TextColumn("value"),
		table.TextColumn("username"),
	}

	t := &GsettingsValues{
		logger:   logger,
		getBytes: execGsettings,
	}

	return table.NewPlugin("gsettings", columns, t.generate)
}

func (t *GsettingsValues) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	users := tablehelpers.GetConstraints(queryContext, "username", tablehelpers.WithAllowedCharacters(allowedCharacters))
	if len(users) < 1 {
		return results, errors.New("gsettings requires at least one username to be specified")
	}
	for _, username := range users {
		var output bytes.Buffer

		err := t.getBytes(ctx, username, &output)
		if err != nil {
			level.Info(t.logger).Log(
				"msg", "error getting bytes for user",
				"username", username,
				"err", err,
			)
			continue
		}

		user_results := t.parse(username, &output)
		results = append(results, user_results...)
	}

	return results, nil
}

// execGsettings writes the output of running 'gsettings' command into the
// supplied bytes buffer
func execGsettings(ctx context.Context, username string, buf *bytes.Buffer) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("finding user by username '%s': %w", username, err)
	}

	cmd := exec.CommandContext(ctx, gsettingsPath, "list-recursively")

	// set the HOME for the the cmd so that gsettings is exec'd properly as the
	// new user.
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", u.HomeDir))

	// Check if the supplied UID is that of the current user
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("checking current user uid: %w", err)
	}

	if u.Uid != currentUser.Uid {
		uid, err := strconv.ParseInt(u.Uid, 10, 32)
		if err != nil {
			return fmt.Errorf("converting uid from string to int: %w", err)
		}
		gid, err := strconv.ParseInt(u.Gid, 10, 32)
		if err != nil {
			return fmt.Errorf("converting gid from string to int: %w", err)
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		}
	}

	dir, err := os.MkdirTemp("", "osq-gsettings")
	if err != nil {
		return fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(dir)

	// if we don't chmod the dir, we get errors like:
	// 'fork/exec /usr/bin/gsettings: permission denied'
	if err := os.Chmod(dir, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	cmd.Dir = dir

	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr
	cmd.Stdout = buf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running gsettings, err is: %s: %w", stderr.String(), err)
	}

	return nil
}

func (t *GsettingsValues) parse(username string, input io.Reader) []map[string]string {
	var results []map[string]string

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			level.Error(t.logger).Log(
				"msg", "unable to process line, not enough segments",
				"line", line,
			)
			continue
		}
		row := make(map[string]string)
		row["schema"] = parts[0]
		row["key"] = parts[1]
		row["value"] = parts[2]
		row["username"] = username

		results = append(results, row)
	}

	if err := scanner.Err(); err != nil {
		level.Debug(t.logger).Log("msg", "scanner error", "err", err)
	}

	return results
}
