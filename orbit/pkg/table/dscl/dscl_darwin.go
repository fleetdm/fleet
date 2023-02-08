//go:build darwin
// +build darwin

// Package dscl allows querying dscl read commands on the local domain.
package dscl

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("command"), // required
		table.TextColumn("path"),    // required
		table.TextColumn("key"),     // required (could be relaxed in the future)
		table.TextColumn("value"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	supportedCommands := []string{"list", "read"}

	getArgumentOpEqual := func(argName string) string {
		argValue := ""
		if constraints, ok := queryContext.Constraints[argName]; ok {
			for _, constraint := range constraints.Constraints {
				if constraint.Operator == table.OperatorEquals {
					argValue = constraint.Expression
				}
			}
		}
		return argValue
	}

	command := getArgumentOpEqual("command")
	if command == "" {
		return nil, fmt.Errorf("missing command argument, supported commands: %+v", supportedCommands)
	}
	supported := false
	for _, supportedCommand := range supportedCommands {
		if supportedCommand == command {
			supported = true
			break
		}
	}
	if !supported {
		return nil, fmt.Errorf("unsupported command: %s, supported commands: %+v", command, supportedCommands)
	}

	path := getArgumentOpEqual("path")
	if path == "" {
		return nil, errors.New("missing path argument")
	}

	key := getArgumentOpEqual("key")
	if key == "" {
		// In the future we can allow this to be empty and return all key/values of a path.
		return nil, errors.New("missing key argument")
	}

	uid, gid, err := common.GetConsoleUidGid()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get console user")
		return nil, fmt.Errorf("failed to get console user: %w", err)
	}

	cmd := exec.Command("dscl", ".", "-"+command, path, key)

	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	value, err := parseDSCLOutput(out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dscl value: %w", err)
	}

	m := []map[string]string{{
		"command": command,
		"path":    path,
		"key":     key,
	}}

	if value != nil {
		m[0]["value"] = *value
	}

	return m, nil
}

func parseDSCLOutput(out []byte) (*string, error) {
	regex := regexp.MustCompile(`(\S):[ \n]([\S\t\f\r\n ]+)`)

	if strings.HasPrefix(string(out), "No such key: ") {
		return nil, nil
	}
	matches := regex.FindSubmatch(out)
	if matches == nil {
		return nil, fmt.Errorf("unexpected entry: %q", string(out))
	}
	value := string(matches[2])
	if value[0] == ' ' {
		value = value[1:]
	}
	return &value, nil
}
