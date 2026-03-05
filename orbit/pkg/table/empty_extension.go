package table

import (
	"context"

	"github.com/osquery/osquery-go/plugin/table"
)

// EmptyExtension implements a basic table extension that displays an instructional
// message to the user during queries.
type EmptyExtension struct {
	name string
	msg  string
}

func NewEmptyExtension(
	name string,
	msg string,
) Opt {
	return WithExtension(
		&EmptyExtension{
			name: name,
			msg:  msg,
		},
	)
}

func (o EmptyExtension) Name() string {
	return o.name
}
func (o EmptyExtension) Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("message"),
	}
}
func (o EmptyExtension) GenerateFunc(_ context.Context, _ table.QueryContext) ([]map[string]string, error) {
	return []map[string]string{{
		"message": o.msg,
	}}, nil
}
