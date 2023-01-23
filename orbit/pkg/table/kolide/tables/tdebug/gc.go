package tdebug

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	gcTableName = "launcher_gc_info"
)

type gcTable struct {
	logger log.Logger
	stats  debug.GCStats
}

func LauncherGcInfo(_client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns()

	t := &gcTable{
		logger: logger,
	}

	return table.NewPlugin(gcTableName, columns, t.generate)
}

func (t *gcTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	debug.ReadGCStats(&t.stats)

	// Make sure the history arrays aren't too large
	if len(t.stats.Pause) > 100 {
		t.stats.Pause = t.stats.Pause[:100]
	}
	if len(t.stats.PauseEnd) > 100 {
		t.stats.PauseEnd = t.stats.PauseEnd[:100]
	}

	for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
		// bounce through json to serialize GCStats
		jsonBytes, err := json.Marshal(t.stats)
		if err != nil {
			return nil, fmt.Errorf("json: %w", err)
		}

		flatData, err := dataflatten.Json(
			jsonBytes,
			dataflatten.WithLogger(t.logger),
			dataflatten.WithQuery(strings.Split(dataQuery, "/")),
		)
		if err != nil {
			level.Info(t.logger).Log("msg", "gc flatten failed", "err", err)
			continue
		}
		results = append(results, dataflattentable.ToMap(flatData, dataQuery, nil)...)
	}
	return results, nil
}
