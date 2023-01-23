package table

// The kolide_ssh_keys table is deprecated in favor of using the
// underlying osquery file table joined against kolide_keyinfo. The
// latter provides more control over the directories we iterate over

import (
	"context"
	"runtime"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/keyidentifier"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

var sshDirs = map[string][]string{
	// "windows": []string{},
}

var sshDirsDefault = []string{".ssh/*"}

type SshKeysTable struct {
	client     *osquery.ExtensionManagerClient
	logger     log.Logger
	kIdentifer *keyidentifier.KeyIdentifier
}

// New returns a new table extension
func SshKeys(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("user"),
		table.TextColumn("path"),
		table.TextColumn("type"),
		table.IntegerColumn("encrypted"),
		table.IntegerColumn("bits"),
		table.TextColumn("fingerprint_sha256"),
		table.TextColumn("fingerprint_md5"),
	}

	// we don't want the logging in osquery, so don't instantiate WithLogger()
	kIdentifer, err := keyidentifier.New()
	if err != nil {
		level.Info(logger).Log(
			"msg", "Failed to create keyidentifier",
			"err", err,
		)
		return nil
	}

	t := &SshKeysTable{
		client:     client,
		logger:     logger,
		kIdentifer: kIdentifer,
	}

	return table.NewPlugin("kolide_ssh_keys", columns, t.generate)
}

func (t *SshKeysTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	// Find the dirs we're going to search
	dirs, ok := sshDirs[runtime.GOOS]
	if !ok {
		dirs = sshDirsDefault
	}

	for _, dir := range dirs {
		files, err := findFileInUserDirs(dir, t.logger)
		if err != nil {
			level.Info(t.logger).Log(
				"msg", "Error finding ssh keys paths",
				"path", dir,
				"err", err,
			)
			continue
		}

		for _, file := range files {
			ki, err := t.kIdentifer.IdentifyFile(file.path)
			if err != nil {
				level.Debug(t.logger).Log(
					"msg", "Failed to get keyinfo for file",
					"file", file.path,
					"err", err,
				)
				continue
			}

			res := map[string]string{
				"path": file.path,
				"user": file.user,
				"type": ki.Type,
			}

			if ki.Encrypted != nil {
				res["encrypted"] = strconv.Itoa(btoi(*ki.Encrypted))
			}

			if ki.Bits != 0 {
				res["bits"] = strconv.FormatInt(int64(ki.Bits), 10)
			}

			if ki.FingerprintSHA256 != "" {
				res["fingerprint_sha256"] = ki.FingerprintSHA256
			}
			if ki.FingerprintMD5 != "" {
				res["fingerprint_md5"] = ki.FingerprintMD5
			}

			results = append(results, res)
		}
	}

	return results, nil
}
