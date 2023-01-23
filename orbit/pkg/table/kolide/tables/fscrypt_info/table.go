package fscrypt_info

import (
	"github.com/go-kit/kit/log"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	tableName = "kolide_fscrypt_info"
)

type Table struct {
	logger log.Logger
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("path"),
		table.IntegerColumn("encrypted"),
		table.TextColumn("mountpoint"),
		table.TextColumn("locked"),
		table.TextColumn("filesystem_type"),
		table.TextColumn("device"),
		table.TextColumn("contents_algo"),
		table.TextColumn("filename_algo"),
	}

	t := &Table{
		logger: logger,
	}
	return table.NewPlugin(tableName, columns, t.generate)
}
