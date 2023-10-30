package dataflattentable

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/kolide/launcher/pkg/dataflatten"
	"github.com/kolide/launcher/pkg/osquery/tables/tablehelpers"
	"github.com/kolide/launcher/pkg/traces"
	"github.com/osquery/osquery-go/plugin/table"
)

type bytesFlattener interface {
	FlattenBytes([]byte, ...dataflatten.FlattenOpts) ([]dataflatten.Row, error)
}

// execTableV2 is the next iteration of the dataflattentable wrapper. Aim to migrate exec based tables to this.
type execTableV2 struct {
	logger         log.Logger
	tableName      string
	flattener      bytesFlattener
	timeoutSeconds int
	tabledebug     bool
	includeStderr  bool
	execPaths      []string
	execArgs       []string
}

type execTableV2Opt func(*execTableV2)

func WithTimeoutSeconds(ts int) execTableV2Opt {
	return func(t *execTableV2) {
		t.timeoutSeconds = ts
	}
}

func WithTableDebug() execTableV2Opt {
	return func(t *execTableV2) {
		t.tabledebug = true
	}
}

func WithAdditionalExecPaths(paths ...string) execTableV2Opt {
	return func(t *execTableV2) {
		t.execPaths = append(t.execPaths, paths...)
	}
}

func WithIncludeStderr() execTableV2Opt {
	return func(t *execTableV2) {
		t.includeStderr = true
	}
}

func NewExecAndParseTable(logger log.Logger, tableName string, p parser, execCmd []string, opts ...execTableV2Opt) *table.Plugin {
	t := &execTableV2{
		logger:         level.NewFilter(log.With(logger, "table", tableName), level.AllowInfo()),
		tableName:      tableName,
		flattener:      flattenerFromParser(p),
		timeoutSeconds: 30,
		execPaths:      execCmd[:1],
		execArgs:       execCmd[1:],
	}

	for _, opt := range opts {
		opt(t)
	}

	if t.tabledebug {
		level.NewFilter(log.With(logger, "table", tableName), level.AllowDebug())
	}

	return table.NewPlugin(t.tableName, Columns(), t.generate)
}

func (t *execTableV2) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	ctx, span := traces.StartSpan(ctx, "table_name", t.tableName)
	defer span.End()

	var results []map[string]string

	execOutput, err := tablehelpers.Exec(ctx, t.logger, t.timeoutSeconds, t.execPaths, t.execArgs, t.includeStderr)
	if err != nil {
		// exec will error if there's no binary, so we never want to record that
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		traces.SetError(span, err)
		level.Info(t.logger).Log("msg", "exec failed", "err", err)
		return nil, nil
	}

	for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
		flattenOpts := []dataflatten.FlattenOpts{
			dataflatten.WithLogger(t.logger),
			dataflatten.WithQuery(strings.Split(dataQuery, "/")),
		}

		flattened, err := t.flattener.FlattenBytes(execOutput, flattenOpts...)
		if err != nil {
			level.Info(t.logger).Log("msg", "failure flattening output", "err", err)
			continue
		}

		results = append(results, ToMap(flattened, dataQuery, nil)...)
	}

	return results, nil
}
