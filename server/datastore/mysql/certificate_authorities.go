package mysql

import (
	"context"
	"encoding/json"
	"fmt"

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
			// TODO EJM wrap all these errors
			return nil, fmt.Errorf("failed to marshal certificate user principal names for new CA %s: %w", ca.Name, err)
		}
	}
	if ca.APIToken != nil {
		encryptedAPIToken, err = encrypt([]byte(*ca.APIToken), ds.serverPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API token for new CA %s: %w", ca.Name, err)
		}
	}
	if ca.Password != nil {
		encryptedPassword, err = encrypt([]byte(*ca.Password), ds.serverPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password for new CA %s: %w", ca.Name, err)
		}
	}
	if ca.Challenge != nil {
		encryptedChallenge, err = encrypt([]byte(*ca.Challenge), ds.serverPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt challenge for new CA %s: %w", ca.Name, err)
		}
	}
	if ca.ClientSecret != nil {
		encryptedClientSecret, err = encrypt([]byte(*ca.ClientSecret), ds.serverPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt client secret for new CA %s: %w", ca.Name, err)
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
	api_token,
	profile_id,
	certificate_common_name,
	certificate_user_principal_names,
	certificate_seat_id,
	admin_url,
	username,
	password,
	challenge,
	client_id,
	client_secret
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new certificate authority %s: %w", ca.Name, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID for new certificate authority %s: %w", ca.Name, err)
	}
	ca.ID = id
	return ca, nil
}
