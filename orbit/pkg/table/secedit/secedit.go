//go:build windows
// +build windows

package secedit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const seceditCmd = "secedit"

type Table struct {
	logger log.Logger
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("mergedpolicy"),
	)

	t := &Table{
		logger: logger,
	}

	return table.NewPlugin("secedit", columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	for _, mergedpolicy := range tablehelpers.GetConstraints(queryContext, "mergedpolicy", tablehelpers.WithDefaults("false")) {
		useMergedPolicy, err := strconv.ParseBool(mergedpolicy)
		if err != nil {
			level.Info(t.logger).Log("msg", "Cannot convert mergedpolicy constraint into a boolean value. Try passing \"true\"", "err", err)
			continue
		}

		for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
			secEditResults, err := t.execSecedit(ctx, useMergedPolicy)
			if err != nil {
				level.Info(t.logger).Log("msg", "secedit failed", "err", err)
				continue
			}

			flatData, err := t.flattenOutput(dataQuery, secEditResults)
			if err != nil {
				level.Info(t.logger).Log("msg", "flatten failed", "err", err)
				continue
			}

			rowData := map[string]string{
				"mergedpolicy": mergedpolicy,
			}

			results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)
		}
	}
	return results, nil
}

func (t *Table) flattenOutput(dataQuery string, systemOutput []byte) ([]dataflatten.Row, error) {
	flattenOpts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(t.logger),
		dataflatten.WithQuery(strings.Split(dataQuery, "/")),
	}

	return dataflatten.Ini(systemOutput, flattenOpts...)
}

func (t *Table) execSecedit(ctx context.Context, mergedPolicy bool) ([]byte, error) {
	// The secedit.exe binary does not support outputting the data we need to stdout
	// Instead we create a tmp directory and pass it to secedit to write the data we need
	// in INI format.
	dir, err := os.MkdirTemp("", "secedit_config")
	if err != nil {
		return nil, fmt.Errorf("creating secedit_config tmp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	dst := filepath.Join(dir, "tmpfile.ini")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	args := []string{"/export", "/cfg", dst}
	if mergedPolicy {
		args = append(args, "/mergedpolicy")
	}

	cmd := exec.CommandContext(ctx, seceditCmd, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	level.Debug(t.logger).Log("msg", "calling secedit", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("calling secedit. Got: %s: %w", stderr.String(), err)
	}

	file, err := os.Open(dst)
	if err != nil {
		return nil, fmt.Errorf("error opening secedit output file: %s: %w", dst, err)
	}
	defer file.Close()

	// By default, secedit outputs files encoded in UTF16 Little Endian. Sadly the Go INI parser
	// cannot read this format by default, therefore we decode the bytes into UTF-8
	rd := transform.NewReader(file, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, fmt.Errorf("error reading secedit output file: %s: %w", dst, err)
	}

	return data, nil
}
