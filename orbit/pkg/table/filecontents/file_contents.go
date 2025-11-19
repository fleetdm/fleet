package filecontents

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

const (
	columnPath     = "path"
	columnContents = "contents"
)

// Columns returns the schema for the file_contents table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn(columnPath),
		table.TextColumn(columnContents),
	}
}

func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	path := ""
	wildcard := false

	if constraintList, present := queryContext.Constraints[columnPath]; present {
		// 'path' is in the where clause
		for _, constraint := range constraintList.Constraints {
			// LIKE
			if constraint.Operator == table.OperatorLike {
				path = constraint.Expression
				wildcard = true
			}
			// =
			if constraint.Operator == table.OperatorEquals {
				path = constraint.Expression
				wildcard = false
			}
		}
	}
	var results []map[string]string
	output, err := processFile(path, wildcard)
	if err != nil {
		return results, err
	}

	for _, item := range output {
		results = append(results, map[string]string{
			columnContents: item.Contents,
			columnPath:     item.Path,
		})
	}

	return results, nil
}

type fileContents struct {
	Contents string
	Path     string
}

func processFile(path string, wildcard bool) ([]fileContents, error) {
	var output []fileContents

	if wildcard {
		replacedPath := strings.ReplaceAll(path, "%", "*")

		files, err := filepath.Glob(replacedPath)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			contents, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}
			output = append(output, fileContents{Path: file, Contents: string(contents)})
		}
	} else {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		output = append(output, fileContents{Path: path, Contents: string(contents)})
	}

	return output, nil
}
