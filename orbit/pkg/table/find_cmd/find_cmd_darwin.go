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
	"bytes"
	"context"
	"errors"
	"fmt"
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

	args := []string{directory}
	if findType != "" {
		args = append(args, "-type", findType)
	}
	if perm != "" {
		args = append(args, "-perm", perm)
	}
	// NUL-terminated output, because file names can contain newlines but never NUL.
	args = append(args, "-print0")

	var stdout bytes.Buffer
	stderr := &cappedBuffer{limit: maxStderrBytes}
	cmd := exec.Command("/usr/bin/find", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("command start failed: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		// We ignore error as these could be of the form:
		// 'find: /System/Volumes/Data/Library/Caches/com.apple.aned: Operation not permitted'
		// which are files unaccessible even for root.
		log.Debug().Err(err).Bytes("stderr", stderr.buf).Msg("find failed")
	}

	rows := []map[string]string{}
	for {
		// No NUL terminator means find died mid-write; drop the partial path.
		path, err := stdout.ReadString(0)
		if err != nil {
			break
		}
		rows = append(rows, map[string]string{
			"directory": directory,
			"perm":      perm,
			"type":      findType,
			"path":      strings.TrimSuffix(path, "\x00"),
		})
	}
	return rows, nil
}

// maxStderrBytes caps how much of find's stderr is kept for the debug log; scans
// without Full Disk Access can print an error for every protected directory.
const maxStderrBytes = 4096

// cappedBuffer keeps the first limit bytes and discards the rest. Writes never
// fail, so exec keeps draining the pipe and find never blocks on stderr.
type cappedBuffer struct {
	buf   []byte
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p[:min(len(p), b.limit-len(b.buf))]...)
	return len(p), nil
}
