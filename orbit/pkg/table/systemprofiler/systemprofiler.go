//go:build darwin
// +build darwin

// Package systemprofiler provides a suite table wrapper around
// `system_profiler` macOS command. It supports some basic arguments
// like `detaillevel` and requested data types.
//
// Note that some detail levels and data types will have performance
// impact if requested.
//
// As the returned data is a complex nested plist, this uses the
// dataflatten tooling. (See
// https://github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten)
//
// Everything, minimal details:
//
//	osquery> select count(*) from system_profiler where datatype like "%" and detaillevel = "mini";
//	+----------+
//	| count(*) |
//	+----------+
//	| 1270     |
//	+----------+
//
// Multiple data types (slightly redacted):
//
//	osquery> select fullkey, key, value, datatype from system_profiler where datatype in ("SPCameraDataType", "SPiBridgeDataType");
//	+----------------------+--------------------+------------------------------------------+-------------------+
//	| fullkey              | key                | value                                    | datatype          |
//	+----------------------+--------------------+------------------------------------------+-------------------+
//	| 0/spcamera_unique-id | spcamera_unique-id | 0x1111111111111111                       | SPCameraDataType  |
//	| 0/_name              | _name              | FaceTime HD Camera                       | SPCameraDataType  |
//	| 0/spcamera_model-id  | spcamera_model-id  | UVC Camera VendorID_1452 ProductID_30000 | SPCameraDataType  |
//	| 0/_name              | _name              | Controller Information                   | SPiBridgeDataType |
//	| 0/ibridge_build      | ibridge_build      | 14Y000                                   | SPiBridgeDataType |
//	| 0/ibridge_model_name | ibridge_model_name | Apple T1 Security Chip                   | SPiBridgeDataType |
//	+----------------------+--------------------+------------------------------------------+-------------------+
// based on github.com/kolide/launcher/pkg/osquery/tables
package systemprofiler

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/groob/plist"
	"github.com/osquery/osquery-go/plugin/table"
)

const systemprofilerPath = "/usr/sbin/system_profiler"

var knownDetailLevels = []string{
	"mini",  // short report (contains no identifying or personal information)
	"basic", // basic hardware and network information
	"full",  // all available information
}

type Property struct {
	Order                string `plist:"_order"`
	SuppressLocalization string `plist:"_suppressLocalization"`
	DetailLevel          string `plist:"_detailLevel"`
}

type Result struct {
	Items          []interface{} `plist:"_items"`
	DataType       string        `plist:"_dataType"`
	SPCommandLine  []string      `plist:"_SPCommandLineArguments"`
	ParentDataType string        `plist:"_parentDataType"`

	// These would be nice to add, but they come back with inconsistent
	// types, so doing a straight unmarshal is hard.
	// DetailLevel    int                 `plist:"_detailLevel"`
	// Properties     map[string]Property `plist:"_properties"`
}

type Table struct {
	logger    log.Logger
	tableName string
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("parentdatatype"),
		table.TextColumn("datatype"),
		table.TextColumn("detaillevel"),
	)

	t := &Table{
		logger:    level.NewFilter(logger, level.AllowInfo()),
		tableName: "system_profiler",
	}

	return table.NewPlugin(t.tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	requestedDatatypes := []string{}

	datatypeQ, ok := queryContext.Constraints["datatype"]
	if !ok || len(datatypeQ.Constraints) == 0 {
		return results, fmt.Errorf("The %s table requires that you specify a constraint for datatype", t.tableName)
	}

	for _, datatypeConstraint := range datatypeQ.Constraints {
		dt := datatypeConstraint.Expression

		// If the constraint is the magic "%", it's eqivlent to an `all` style
		if dt == "%" {
			requestedDatatypes = []string{}
			break
		}

		requestedDatatypes = append(requestedDatatypes, dt)
	}

	var detailLevel string
	if q, ok := queryContext.Constraints["detaillevel"]; ok && len(q.Constraints) != 0 {
		if len(q.Constraints) > 1 {
			level.Info(t.logger).Log("msg", "WARNING: Only using the first detaillevel request")
		}

		dl := q.Constraints[0].Expression
		for _, known := range knownDetailLevels {
			if known == dl {
				detailLevel = dl
			}
		}

	}

	systemProfilerOutput, err := t.execSystemProfiler(ctx, detailLevel, requestedDatatypes)
	if err != nil {
		return results, fmt.Errorf("exec: %w", err)
	}

	if q, ok := queryContext.Constraints["query"]; ok && len(q.Constraints) != 0 {
		for _, constraint := range q.Constraints {
			dataQuery := constraint.Expression
			results = append(results, t.getRowsFromOutput(dataQuery, detailLevel, systemProfilerOutput)...)
		}
	} else {
		results = append(results, t.getRowsFromOutput("", detailLevel, systemProfilerOutput)...)
	}

	return results, nil
}

func (t *Table) getRowsFromOutput(dataQuery, detailLevel string, systemProfilerOutput []byte) []map[string]string {
	var results []map[string]string

	flattenOpts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(t.logger),
		dataflatten.WithQuery(strings.Split(dataQuery, "/")),
	}

	var systemProfilerResults []Result
	if err := plist.Unmarshal(systemProfilerOutput, &systemProfilerResults); err != nil {
		level.Info(t.logger).Log("msg", "error unmarshalling system_profile output", "err", err)
		return nil
	}

	for _, systemProfilerResult := range systemProfilerResults {

		dataType := systemProfilerResult.DataType

		flatData, err := dataflatten.Flatten(systemProfilerResult.Items, flattenOpts...)
		if err != nil {
			level.Info(t.logger).Log("msg", "failure flattening system_profile output", "err", err)
			continue
		}

		rowData := map[string]string{
			"datatype":       dataType,
			"parentdatatype": systemProfilerResult.ParentDataType,
			"detaillevel":    detailLevel,
		}

		results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)
	}

	return results
}

func (t *Table) execSystemProfiler(ctx context.Context, detailLevel string, subcommands []string) ([]byte, error) {
	timeout := 45 * time.Second
	if detailLevel == "full" {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	args := []string{"-xml"}

	if detailLevel != "" {
		args = append(args, "-detailLevel", detailLevel)
	}

	args = append(args, subcommands...)

	cmd := exec.CommandContext(ctx, systemprofilerPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	level.Debug(t.logger).Log("msg", "calling system_profiler", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("calling system_profiler. Got: %s: %w", string(stderr.Bytes()), err)
	}

	return stdout.Bytes(), nil
}
