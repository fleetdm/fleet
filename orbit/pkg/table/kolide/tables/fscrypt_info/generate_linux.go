//go:build linux
// +build linux

package fscrypt_info

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	allowedCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/-_.+"
)

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	paths := tablehelpers.GetConstraints(queryContext, "path", tablehelpers.WithAllowedCharacters(allowedCharacters))
	if len(paths) < 1 {
		return nil, errors.New(tableName + " requires at least one path to be specified")
	}

	results := make([]map[string]string, len(paths))

	for i, dirpath := range paths {
		info, err := GetInfo(dirpath)
		if err != nil {
			level.Info(t.logger).Log(
				"msg", "error getting fscrypt info",
				"path", dirpath,
				"err", err,
			)
			continue
		}
		results[i] = map[string]string{
			"path":            dirpath,
			"encrypted":       boolToRow(info.Encrypted),
			"locked":          info.Locked,
			"mountpoint":      info.Mountpoint,
			"filesystem_type": info.FilesystemType,
			"device":          info.Device,
			"contents_algo":   info.ContentsAlgo,
			"filename_algo":   info.FilenameAlgo,
		}
	}

	return results, nil
}

func boolToRow(v bool) string {
	if v {
		return "1"
	}

	return "0"
}
