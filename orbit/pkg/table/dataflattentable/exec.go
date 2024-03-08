// based on github.com/kolide/launcher/pkg/osquery/tables
package dataflattentable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

type ExecTableOpt func(*Table)

// WithKVSeparator sets the delimiter between key and value. It replaces the
// default ":" in dataflattentable.Table
func WithKVSeparator(separator string) ExecTableOpt {
	return func(t *Table) {
		t.keyValueSeparator = separator
	}
}

func WithBinDirs(binDirs ...string) ExecTableOpt {
	return func(t *Table) {
		t.binDirs = binDirs
	}
}

func TablePluginExec(logger log.Logger, tableName string, dataSourceType DataSourceType, execArgs []string, opts ...ExecTableOpt) *table.Plugin {
	columns := Columns()

	t := &Table{
		logger:            level.NewFilter(logger, level.AllowInfo()),
		tableName:         tableName,
		execArgs:          execArgs,
		keyValueSeparator: ":",
	}

	for _, opt := range opts {
		opt(t)
	}

	switch dataSourceType {
	case PlistType:
		t.flattenBytesFunc = dataflatten.Plist
	case JsonType:
		t.flattenBytesFunc = dataflatten.Json
	case KeyValueType:
		// TODO: allow callers of TablePluginExec to specify the record
		// splitting strategy
		t.flattenBytesFunc = dataflatten.StringDelimitedFunc(t.keyValueSeparator, dataflatten.DuplicateKeys)
	default:
		panic("Unknown data source type")
	}

	return table.NewPlugin(t.tableName, columns, t.generateExec)
}

func (t *Table) generateExec(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	execBytes, err := t.exec(ctx)
	if err != nil {
		// exec will error if there's no binary, so we never want to record that
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		// If the exec failed for some reason, it's probably better to return no results, and log the,
		// error. Returning an error here will cause a table failure, and thus break joins
		level.Info(t.logger).Log("msg", "failed to exec", "err", err)
		return nil, nil
	}

	for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
		flattenOpts := []dataflatten.FlattenOpts{
			dataflatten.WithLogger(t.logger),
			dataflatten.WithQuery(strings.Split(dataQuery, "/")),
		}

		flattened, err := t.flattenBytesFunc(execBytes, flattenOpts...)
		if err != nil {
			level.Info(t.logger).Log("msg", "failure flattening output", "err", err)
			continue
		}

		results = append(results, ToMap(flattened, dataQuery, nil)...)
	}

	return results, nil
}

func (t *Table) exec(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	possibleBinaries := []string{}

	if t.binDirs == nil || len(t.binDirs) == 0 {
		possibleBinaries = []string{t.execArgs[0]}
	} else {
		for _, possiblePath := range t.binDirs {
			possibleBinaries = append(possibleBinaries, filepath.Join(possiblePath, t.execArgs[0]))
		}
	}

	for _, execPath := range possibleBinaries {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd := exec.CommandContext(ctx, execPath, t.execArgs[1:]...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		level.Debug(t.logger).Log("msg", "calling %s", "args", cmd.String())

		if err := cmd.Run(); os.IsNotExist(err) {
			// try the next binary
			continue
		} else if err != nil {
			return nil, fmt.Errorf("calling %s. Got: %s: %w", t.execArgs[0], string(stderr.Bytes()), err)
		}

		// success!
		return stdout.Bytes(), nil
	}

	// None of the possible execs were found
	return nil, fmt.Errorf("Unable to exec '%s'. No binary found is specified paths", t.execArgs[0])
}
