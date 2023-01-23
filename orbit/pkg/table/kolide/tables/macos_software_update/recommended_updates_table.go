//go:build darwin
// +build darwin

package macos_software_update

/*
#cgo darwin CFLAGS: -DDARWIN -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa
#include "sus.h"
*/
import (
	"C"
)

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

var updatesData []map[string]interface{}

type Table struct {
	logger log.Logger
}

func RecommendedUpdates(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns()

	tableName := "kolide_macos_recommended_updates"

	t := &Table{
		logger: log.With(logger, "table", tableName),
	}

	return table.NewPlugin(tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	data := getUpdates()

	for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
		flattened, err := dataflatten.Flatten(data, dataflatten.WithLogger(t.logger), dataflatten.WithQuery(strings.Split(dataQuery, "/")))
		if err != nil {
			level.Info(t.logger).Log("msg", "Error flattening data", "err", err)
			return nil, nil
		}
		results = append(results, dataflattentable.ToMap(flattened, dataQuery, nil)...)
	}

	return results, nil
}

//export updatesFound
func updatesFound(numUpdates C.uint) {
	// getRecommendedUpdates will use this callback to indicate how many updates have been found
	updatesData = make([]map[string]interface{}, numUpdates)
}

//export updateKeyValueFound
func updateKeyValueFound(index C.uint, key, value *C.char) {
	// getRecommendedUpdates will use this callback for each key-value found
	if updatesData[index] == nil {
		updatesData[index] = make(map[string]interface{})
	}
	updatesData[index][C.GoString(key)] = C.GoString(value)
}

func getUpdates() map[string]interface{} {
	results := make(map[string]interface{})

	// Since updatesData is package level, reset it before each invocation to purge stale results
	updatesData = nil

	C.getRecommendedUpdates()
	results["RecommendedUpdates"] = updatesData

	return results
}
