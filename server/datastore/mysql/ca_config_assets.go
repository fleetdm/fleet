package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetAllCAConfigAssetsByType(ctx context.Context, assetType fleet.CAConfigAssetType) (map[string]fleet.CAConfigAsset, error) {
	stmt := `
SELECT
	   name, type, value
FROM
	  ca_config_assets
WHERE
	  type = ?
		`

	var res []fleet.CAConfigAsset
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, assetType); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get CA config assets")
	}

	if len(res) == 0 {
		return nil, notFound("CAConfigAsset")
	}

	assetMap := make(map[string]fleet.CAConfigAsset, len(res))
	for _, asset := range res {
		decryptedVal, err := decrypt(asset.Value, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "decrypting CA config asset %s", asset.Name)
		}
		asset.Value = decryptedVal
		assetMap[asset.Name] = asset
	}

	return assetMap, nil
}

func (ds *Datastore) SaveCAConfigAssets(ctx context.Context, assets []fleet.CAConfigAsset) error {
	return ds.saveCAConfigAssets(ctx, ds.writer(ctx), assets)
}

func (ds *Datastore) saveCAConfigAssets(ctx context.Context, tx sqlx.ExtContext, assets []fleet.CAConfigAsset) error {
	if len(assets) == 0 {
		return nil
	}

	stmt := fmt.Sprintf(`
	INSERT INTO ca_config_assets (name, type, value)
	VALUES %s
	ON DUPLICATE KEY UPDATE
		value = VALUES(value),
		type = VALUES(type)
	`, strings.TrimSuffix(strings.Repeat("(?,?,?),", len(assets)), ","))

	args := make([]interface{}, 0, len(assets)*3)
	for _, asset := range assets {
		encryptedVal, err := encrypt(asset.Value, ds.serverPrivateKey)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "encrypting CA config asset %s", asset.Name)
		}
		args = append(args, asset.Name, asset.Type, encryptedVal)
	}

	_, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save CA config assets")
	}
	return nil
}

func (ds *Datastore) GetCAConfigAsset(ctx context.Context, name string, assetType fleet.CAConfigAssetType) (*fleet.CAConfigAsset, error) {
	stmt := `
	SELECT
		name, type, value
	FROM
		ca_config_assets
	WHERE
		name = ? AND type = ?
	`

	var asset fleet.CAConfigAsset
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &asset, stmt, name, assetType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("CAConfigAsset").WithName(name)
		}
		return nil, ctxerr.Wrapf(ctx, err, "get CA config asset %s", name)
	}

	decryptedVal, err := decrypt(asset.Value, ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "decrypting CA config asset %s", asset.Name)
	}
	asset.Value = decryptedVal

	return &asset, nil
}

func (ds *Datastore) DeleteCAConfigAssets(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}

	stmt := `
		DELETE FROM ca_config_assets
		WHERE name IN (?)
	`
	stmt, args, err := sqlx.In(stmt, names)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query for deleting CA config assets")
	}

	_, err = ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete CA config assets")
	}

	return nil
}
