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
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

var (
	productsData []map[string]interface{}
	cachedTime   time.Time
)

func AvailableProducts(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns()

	tableName := "kolide_macos_available_products"

	t := &Table{
		logger: log.With(logger, "table", tableName),
	}

	return table.NewPlugin(tableName, columns, t.generateAvailableProducts)
}

func (t *Table) generateAvailableProducts(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	data := getProducts()

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

//export productsFound
func productsFound(numProducts C.uint) {
	// getAvailableProducts will use this callback to indicate how many products have been found
	productsData = make([]map[string]interface{}, numProducts)
}

//export productKeyValueFound
func productKeyValueFound(index C.uint, key, value *C.char) {
	// getAvailableProducts will use this callback for each key-value found
	if productsData[index] == nil {
		productsData[index] = make(map[string]interface{})
	}
	if value != nil {
		productsData[index][C.GoString(key)] = C.GoString(value)
	}
}

//export productNestedKeyValueFound
func productNestedKeyValueFound(index C.uint, parent, key, value *C.char) {
	// getAvailableProducts will use this callback for each nested key-value found
	if productsData[index] == nil {
		productsData[index] = make(map[string]interface{})
	}

	parentStr := C.GoString(parent)
	if productsData[index][parentStr] == nil {
		productsData[index][parentStr] = make(map[string]interface{})
	}

	if value != nil {
		parentObj, _ := productsData[index][parentStr].(map[string]interface{})
		parentObj[C.GoString(key)] = C.GoString(value)
	}
}

func getProducts() map[string]interface{} {
	results := make(map[string]interface{})

	// Calling getAvailableProducts is an expensive operation and could cause performance
	// problems if called too frequently. Here we cache the data and restrict the
	// frequency of invocations to at most once per minute.
	if productsData != nil && time.Since(cachedTime) < 1*time.Minute {
		results["AvailableProducts"] = productsData
		return results
	}

	// Since productsData is package level, reset it before each invocation to purge stale results
	productsData = nil

	C.getAvailableProducts()

	// Remember when we last retrieved the data
	cachedTime = time.Now()

	results["AvailableProducts"] = productsData
	return results
}
