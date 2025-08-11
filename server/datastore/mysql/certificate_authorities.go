package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetCertificateAuthorityByID(ctx context.Context, id uint) (*fleet.CertificateAuthority, error) {
	stmt := `
	SELECT
		id, type, name, url, api_token, profile_id,
		certificate_common_name, certificate_user_principal_names,
		certificate_seat_id, admin_url, username, client_id,
		client_secret, password, challenge
	FROM
		certificate_authorities
	WHERE
		id = ?
	`

	var ca fleet.CertificateAuthority
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &ca, stmt, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("CertificateAuthority").WithID(id)
		}
		return nil, ctxerr.Wrapf(ctx, err, "get CertificateAuthority %d", id)
	}

	return &ca, nil
}
