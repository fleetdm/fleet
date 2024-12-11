//go:build darwin
// +build darwin

// Package find_cmd implements a table that executes the /usr/bin/find command.
// This table provides only a subset of the find functionality. Currently only
// allows setting the -perm and -type arguments.
//
// NOTE(lucas): Why does this table exist?
// Initially we implemented queries that used the osquery core `file` table,
// but when processing a high number (10k+) of files it exceeded osquery
// default CPU and memory limits.
package find_cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		// directory is the first argument of the find command (basically
		// where to search for files).
		table.TextColumn("directory"),
		// type allows setting find's '-type' argument.
		table.TextColumn("type"),
		// perm allows setting find's '-perm' argument.
		table.TextColumn("perm"),
		// not_perm allows setting find's '-not -perm' argument.
		table.TextColumn("not_perm"),
		// mindepth allows setting find's '-mindepth' argument.
		table.TextColumn("mindepth"),
		// maxdepth allows setting find's '-maxdepth' argument.
		table.TextColumn("maxdepth"),
		// path are the found directories.
		table.TextColumn("path"),
	}
}

var permRegexp = regexp.MustCompile(`[-+]*\d+`)

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
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

	directory := getArgumentOpEqual("directory")
	if directory == "" {
		return nil, errors.New("missing directory argument")
	}
	if !filepath.IsAbs(directory) {
		return nil, errors.New("directory must be an absolute path")
	}

	findType := getArgumentOpEqual("type")
	if findType != "" {
		switch findType {
		case "b", "c", "d", "f", "l", "p", "s":
			// OK
		default:
			return nil, errors.New("type must be one of: 'b', 'c', 'd', 'f', 'l', 'p' or 's'")
		}
	}

	perm := getArgumentOpEqual("perm")
	if perm != "" {
		if !permRegexp.Match([]byte(perm)) {
			return nil, fmt.Errorf("perm must be of the form: %s", permRegexp)
		}
	}

	notPerm := getArgumentOpEqual("not_perm")
	if notPerm != "" {
		if !permRegexp.Match([]byte(notPerm)) {
			return nil, fmt.Errorf("not_perm must be of the form: %s", permRegexp)
		}
	}

	minDepth := getArgumentOpEqual("mindepth")
	maxDepth := getArgumentOpEqual("maxdepth")

	args := []string{directory}
	if findType != "" {
		args = append(args, "-type", findType)
	}
	if perm != "" {
		args = append(args, "-perm", perm)
	}
	if notPerm != "" {
		args = append(args, "-not", "-perm", notPerm)
	}
	if minDepth != "" {
		args = append(args, "-mindepth", minDepth)
	}
	if maxDepth != "" {
		args = append(args, "-maxdepth", maxDepth)
	}

	cmd := exec.Command("/usr/bin/find", args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("command start failed: %w", err)
	}

	var outDirs []string
	reader := bufio.NewReader(stdoutPipe)
	line, err := reader.ReadString('\n')
	for err == nil {
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			continue
		}
		outDirs = append(outDirs, line)
		line, err = reader.ReadString('\n')
	}
	if err != io.EOF {
		return nil, fmt.Errorf("unexpected error: %w", err)
	}

	stderr, err := io.ReadAll(stderrPipe)
	if err != nil {
		log.Debug().Err(err).Msg("failed to read find stderr")
	}

	if err := cmd.Wait(); err != nil {
		// We ignore error as these could be of the form:
		// 'find: /System/Volumes/Data/Library/Caches/com.apple.aned: Operation not permitted'
		// which are files unaccessible even for root.
		log.Debug().Err(err).Bytes("stderr", stderr).Msg("find failed")
	}

	rows := make([]map[string]string, 0, len(outDirs))
	for _, outDir := range outDirs {
		rows = append(rows, map[string]string{
			"directory": directory,
			"perm":      perm,
			"not_perm":  notPerm,
			"type":      findType,
			"mindepth":  minDepth,
			"maxdepth":  maxDepth,
			"path":      outDir,
		})
	}
	return rows, nil
}
