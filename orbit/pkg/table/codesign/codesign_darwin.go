//go:build darwin
// +build darwin

// Package codesign implements an extension osquery table
// to get signature information of macOS applications.
package codesign

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		// path is the absolute path to the app bundle.
		// It's required and only supports the equality operator.
		table.TextColumn("path"),
		// team_identifier is the "Team ID", aka "Signature ID", "Developer ID".
		// The value is "" if the app doesn't have a team identifier set.
		// (this is the case for example for builtin Apple apps).
		//
		// See https://developer.apple.com/help/account/manage-your-team/locate-your-team-id/.
		table.TextColumn("team_identifier"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	constraints, ok := queryContext.Constraints["path"]
	if !ok || len(constraints.Constraints) == 0 {
		return nil, errors.New("missing path")
	}

	var paths []string
	for _, constraint := range constraints.Constraints {
		if constraint.Operator != table.OperatorEquals {
			return nil, errors.New("only supported operator for 'path' is '='")
		}
		paths = append(paths, constraint.Expression)
	}

	var rows []map[string]string
	for _, path := range paths {
		row := map[string]string{
			"path":            path,
			"team_identifier": "",
		}
		output, err := exec.CommandContext(ctx, "/usr/bin/codesign",
			// `codesign --display` does not perform any verification of executables/resources,
			// it just parses and displays signature information read from the `Contents` folder.
			"--display",
			// If we don't set verbose it only prints the executable path.
			"--verbose",
			path,
		).CombinedOutput() // using CombinedOutput because output is in stderr and stdout is empty.
		if err != nil {
			// Logging as debug to prevent non signed apps to generate a lot of logged errors.
			log.Debug().Err(err).Str("output", string(output)).Str("path", path).Msg("codesign --display failed")
			rows = append(rows, row)
			continue
		}
		info := parseCodesignOutput(output)
		row["team_identifier"] = info.teamIdentifier
		rows = append(rows, row)
	}

	return rows, nil
}

type parsedInfo struct {
	teamIdentifier string
}

func parseCodesignOutput(output []byte) parsedInfo {
	const teamIdentifierPrefix = "TeamIdentifier="

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var info parsedInfo
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, teamIdentifierPrefix) {
			info.teamIdentifier = strings.TrimSpace(strings.TrimPrefix(line, teamIdentifierPrefix))
			// "not set" is usually displayed on Apple builtin apps.
			if info.teamIdentifier == "not set" {
				info.teamIdentifier = ""
			}
		}
	}
	return info
}
