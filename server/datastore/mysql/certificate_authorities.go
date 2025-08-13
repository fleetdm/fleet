package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

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
		if strings.Contains(err.Error(), "idx_ca_type_name") {
			return nil, fleet.ConflictError{Message: "a certificate authority with this name already exists"}
		}
		return nil, ctxerr.Wrap(ctx, err, "inserting new certificate authority")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last insert ID for new certificate authority")
	}
	ca.ID = uint(id) //nolint:gosec // dismiss G115
	return ca, nil
}

func (ds *Datastore) DeleteCertificateAuthority(ctx context.Context, certificateAuthorityID uint) (*fleet.CertificateAuthority, error) {
	// TODO: Get the certificate before deleting to return it so we can create an activity event.
	stmt := "DELETE FROM certificate_authorities WHERE id = ?"
	result, err := ds.writer(ctx).ExecContext(ctx, stmt, certificateAuthorityID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("deleting certificate authority with id %d", certificateAuthorityID))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting rows affected by delete certificate authority")
	}

	if rowsAffected < 1 {
		return nil, common_mysql.NotFound(fmt.Sprintf("certificate authority with id %d", certificateAuthorityID))
	}

	return nil, nil
}
