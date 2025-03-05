package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetAllCAConfigAssets(ctx context.Context) (map[string]fleet.CAConfigAsset, error) {
	stmt := `
SELECT
    name, type, value
FROM
   ca_config_assets
	`

	var res []fleet.CAConfigAsset
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt); err != nil {
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
