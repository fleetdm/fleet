package table

import (
	"context"
	"debug/macho"
	"errors"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

func MachoInfo() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("path"),
		table.TextColumn("name"),
		table.TextColumn("cpu"),
	}

	return table.NewPlugin("kolide_macho_info", columns, generateMacho)
}

func generateMacho(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	q, ok := queryContext.Constraints["path"]
	if !ok || len(q.Constraints) == 0 {
		return nil, errors.New("The kolide_macho_info table requires that you specify a constraint WHERE path =")
	}
	path := q.Constraints[0].Expression
	f, err := macho.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []map[string]string
	results = append(results, map[string]string{
		"path": path,
		"name": appFromPath(path),
		"cpu":  f.Cpu.String(),
	})
	return results, nil
}

func appFromPath(path string) string {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasSuffix(part, ".app") {
			return part
		}
	}

	return ""
}
