package mysql

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// ReplaceAppleSoftwareUpdateAssets upserts GDMF assets for class and deletes
// rows that are no longer present. Existing rows keep first_seen_at.
func (ds *Datastore) ReplaceAppleSoftwareUpdateAssets(ctx context.Context, class fleet.AppleSoftwareUpdateAssetClass, assets []fleet.AppleSoftwareUpdateAsset) error {
	if len(assets) == 0 {
		return ctxerr.New(ctx, "refusing to replace apple software update assets with empty set")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		keep := map[string]struct{}{}
		for _, a := range assets {
			devices := a.SupportedDevices
			if len(devices) == 0 || !json.Valid(devices) {
				devices = []byte("[]")
			}
			_, err := tx.ExecContext(ctx, `
INSERT INTO apple_software_update_assets
  (class, product_version, build, posting_date, expiration_date, supported_devices)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  posting_date = VALUES(posting_date),
  expiration_date = VALUES(expiration_date),
  supported_devices = VALUES(supported_devices),
  updated_at = CURRENT_TIMESTAMP(6)
`, class, a.ProductVersion, a.Build, a.PostingDate, a.ExpirationDate, devices)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "upsert apple software update asset")
			}
			keep[a.ProductVersion+"\x00"+a.Build] = struct{}{}
		}

		type existing struct {
			ID             uint   `db:"id"`
			ProductVersion string `db:"product_version"`
			Build          string `db:"build"`
		}
		var rows []existing
		if err := sqlx.SelectContext(ctx, tx, &rows, `
SELECT id, product_version, build FROM apple_software_update_assets WHERE class = ?
`, class); err != nil {
			return ctxerr.Wrap(ctx, err, "list apple software update assets for prune")
		}
		var deleteIDs []uint
		for _, r := range rows {
			if _, ok := keep[r.ProductVersion+"\x00"+r.Build]; !ok {
				deleteIDs = append(deleteIDs, r.ID)
			}
		}
		if len(deleteIDs) == 0 {
			return nil
		}
		stmt, args, err := sqlx.In(`DELETE FROM apple_software_update_assets WHERE id IN (?)`, deleteIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build prune apple software update assets")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "prune apple software update assets")
		}
		return nil
	})
}
