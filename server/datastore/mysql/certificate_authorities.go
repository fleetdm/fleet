package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
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

	// return &fleet.CertificateAuthority{
	// 	ID:                            id,
	// 	Type:                          "digicert", // Example type, adjust as needed
	// 	Name:                          "Example DigiCert CA",
	// 	URL:                           "https://example.com",
	// 	APIToken:                      ptr.String("example-token"),
	// 	ProfileID:                     ptr.String("example-profile-id"),
	// 	CertificateCommonName:         ptr.String("example.com"),
	// 	CertificateUserPrincipalNames: []string{"user@example.com"},
	// 	CertificateSeatID:             ptr.String("example-seat-id"),
	// 	CreatedAt:                     time.Now(),
	// 	UpdatedAt:                     time.Now(),
	// }, nil

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

	// return []*fleet.CertificateAuthoritySummary{
	// 	{ID: 1, Name: "Example CA", Type: "digicert"},
	// 	{ID: 2, Name: "Example CA 2", Type: "hydrant"},
	// 	{ID: 3, Name: "Example CA 3", Type: "ndes_scep_proxy"},
	// 	{ID: 4, Name: "Example CA 4", Type: "custom_scep_proxy"},
	// }, nil

	var cas []*fleet.CertificateAuthoritySummary
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cas, stmt); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "list CertificateAuthorities")
	}

	return cas, nil
}

// Create CA. MUST include secrets
func (ds *Datastore) NewCertificateAuthority(ctx context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
	var upns []byte
	var encryptedPassword []byte
	var encryptedChallenge []byte
	var encryptedAPIToken []byte
	var encryptedClientSecret []byte
	var err error
	if ca.CertificateUserPrincipalNames != nil {
		upns, err = json.Marshal(ca.CertificateUserPrincipalNames)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "marshalling certificate user principal names for new certificate authority")
		}
	}
	if ca.APIToken != nil {
		encryptedAPIToken, err = encrypt([]byte(*ca.APIToken), ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "encrypting API token for new certificate authority")
		}
	}
	if ca.Password != nil {
		encryptedPassword, err = encrypt([]byte(*ca.Password), ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "encrypting password for new certificate authority")
		}
	}
	if ca.Challenge != nil {
		encryptedChallenge, err = encrypt([]byte(*ca.Challenge), ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "encrypting challenge for new certificate authority")
		}
	}
	if ca.ClientSecret != nil {
		encryptedClientSecret, err = encrypt([]byte(*ca.ClientSecret), ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "encrypting client secret for new certificate authority")
		}
	}
	args := []interface{}{
		ca.Type,
		ca.Name,
		ca.URL,
		encryptedAPIToken,
		ca.ProfileID,
		ca.CertificateCommonName,
		upns,
		ca.CertificateSeatID,
		ca.AdminURL,
		ca.Username,
		encryptedPassword,
		encryptedChallenge,
		ca.ClientID,
		encryptedClientSecret,
	}
	stmt := `INSERT INTO certificate_authorities (
	type,
	name,
	url,
	api_token_encrypted,
	profile_id,
	certificate_common_name,
	certificate_user_principal_names,
	certificate_seat_id,
	admin_url,
	username,
	password_encrypted,
	challenge_encrypted,
	client_id,
	client_secret_encrypted
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting new certificate authority")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last insert ID for new certificate authority")
	}
	ca.ID = uint(id) //nolint:gosec // dismiss G115
	return ca, nil
}
