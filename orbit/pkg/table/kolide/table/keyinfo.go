package table

import (
	"context"
	"errors"
	"strconv"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/keyidentifier"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	osquery "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

type KeyInfoTable struct {
	client     *osquery.ExtensionManagerClient
	logger     log.Logger
	kIdentifer *keyidentifier.KeyIdentifier
}

func KeyInfo(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
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

	t := &KeyInfoTable{
		client:     client,
		logger:     logger,
		kIdentifer: kIdentifer,
	}

	return table.NewPlugin("kolide_keyinfo", columns, t.generate)
}

func (t *KeyInfoTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	q, ok := queryContext.Constraints["path"]
	if !ok || len(q.Constraints) == 0 {
		return results, errors.New("The kolide_keyinfo table requires that you specify a constraint for path")
	}

	for _, constraint := range q.Constraints {
		ki, err := t.kIdentifer.IdentifyFile(constraint.Expression)
		if err != nil {
			level.Debug(t.logger).Log(
				"msg", "Failed to get keyinfo for file",
				"file", constraint.Expression,
				"err", err,
			)
			continue
		}

		res := map[string]string{
			"path": constraint.Expression,
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

	return results, nil
}
