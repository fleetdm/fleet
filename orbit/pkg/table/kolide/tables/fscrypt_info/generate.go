//go:build !linux
// +build !linux

package fscrypt_info

import (
	"context"
	"errors"
	"runtime"

	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	level.Info(t.logger).Log(
		"msg", tableName+" is only supported on linux",
		"goos", runtime.GOOS,
	)
	return nil, errors.New("Platform Unsupported")
}
