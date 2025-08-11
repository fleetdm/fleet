package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
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

	return &fleet.CertificateAuthority{
		ID:                            id,
		Type:                          "digicert", // Example type, adjust as needed
		Name:                          "Example DigiCert CA",
		URL:                           "https://example.com",
		APIToken:                      ptr.String("example-token"),
		ProfileID:                     ptr.String("example-profile-id"),
		CertificateCommonName:         ptr.String("example.com"),
		CertificateUserPrincipalNames: []string{"user@example.com"},
		CertificateSeatID:             ptr.String("example-seat-id"),
		CreatedAt:                     time.Now(),
		UpdatedAt:                     time.Now(),
	}, nil

	var ca fleet.CertificateAuthority
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &ca, stmt, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("CertificateAuthority").WithID(id)
		}
		return nil, ctxerr.Wrapf(ctx, err, "get CertificateAuthority %d", id)
	}

	return &ca, nil
}

func (ds *Datastore) ListCertificateAuthorities(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
	stmt := `
	SELECT
		id, name, type
	FROM
		certificate_authorities
	ORDER BY
		name
	`

	var cas []*fleet.CertificateAuthoritySummary
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cas, stmt); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "list CertificateAuthorities")
	}

	return cas, nil
}
