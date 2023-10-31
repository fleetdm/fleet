//go:build linux
// +build linux

// based on github.com/kolide/launcher/pkg/osquery/tables
package xrdb

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

var xrdbPath = "/usr/bin/xrdb"

const (
	allowedUsernameCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-."
	allowedDisplayCharacters  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789:."
)

type execer func(ctx context.Context, display, username string, buf *bytes.Buffer) error

type XRDBSettings struct {
	logger   log.Logger
	getBytes execer
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("key"),
		table.TextColumn("value"),
		table.TextColumn("display"),
		table.TextColumn("username"),
	}

	t := &XRDBSettings{
		logger:   logger,
		getBytes: execXRDB,
	}

	return table.NewPlugin("xrdb", columns, t.generate)
}

func (t *XRDBSettings) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	users := tablehelpers.GetConstraints(queryContext, "username", tablehelpers.WithAllowedCharacters(allowedUsernameCharacters))
	if len(users) < 1 {
		return results, errors.New("xrdb requires at least one username to be specified")
	}

	displays := tablehelpers.GetConstraints(queryContext, "display",
		tablehelpers.WithAllowedCharacters(allowedDisplayCharacters),
		tablehelpers.WithDefaults(":0"),
	)
	for _, username := range users {
		for _, display := range displays {
			var output bytes.Buffer

			err := t.getBytes(ctx, display, username, &output)
			if err != nil {
				level.Info(t.logger).Log(
					"msg", "error getting bytes for user",
					"username", username,
					"err", err,
				)
				continue
			}
			user_results := t.parse(display, username, &output)
			results = append(results, user_results...)
		}
	}

	return results, nil
}

// execXRDB writes the output of running 'xrdb' command into the
// supplied bytes buffer
func execXRDB(ctx context.Context, displayNum, username string, buf *bytes.Buffer) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("finding user by username '%s': %w", username, err)
	}

	cmd := exec.CommandContext(ctx, xrdbPath, "-display", displayNum, "-global", "-query")

	// set the HOME cmd so that xrdb is exec'd properly as the new user.
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

	dir, err := os.MkdirTemp("", "osq-xrdb")
	if err != nil {
		return fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := os.Chmod(dir, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	cmd.Dir = dir
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr
	cmd.Stdout = buf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running xrdb, err is: %s: %w", stderr.String(), err)
	}

	return nil
}

func (t *XRDBSettings) parse(display, username string, input io.Reader) []map[string]string {
	var results []map[string]string

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			level.Error(t.logger).Log(
				"msg", "unable to process line, not enough segments",
				"line", line,
			)
			continue
		}
		row := make(map[string]string)
		row["key"] = parts[0]
		row["value"] = strings.TrimSpace(parts[1])
		row["display"] = display
		row["username"] = username

		results = append(results, row)
	}

	if err := scanner.Err(); err != nil {
		level.Debug(t.logger).Log("msg", "scanner error", "err", err)
	}

	return results
}
