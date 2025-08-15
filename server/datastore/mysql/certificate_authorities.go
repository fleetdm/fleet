package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

type certificateAuthorityWithEncryptedSecrets struct {
	fleet.CertificateAuthority
	APITokenEncrypted                []byte `db:"api_token_encrypted"`
	PasswordEncrypted                []byte `db:"password_encrypted"`
	ChallengeEncrypted               []byte `db:"challenge_encrypted"`
	ClientSecretEncrypted            []byte `db:"client_secret_encrypted"`
	CertificateUserPrincipalNamesRaw []byte `db:"certificate_user_principal_names"`
}

func (ds *Datastore) GetCertificateAuthorityByID(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
	stmt := `
	SELECT
		id,
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
		FROM
			certificate_authorities
		WHERE
			id = ?
		`

	var ca certificateAuthorityWithEncryptedSecrets
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &ca, stmt, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("CertificateAuthority").WithID(id)
		}
		return nil, ctxerr.Wrapf(ctx, err, "get CertificateAuthority %d", id)
	}

	if ca.CertificateUserPrincipalNamesRaw != nil {
		if err := json.Unmarshal(ca.CertificateUserPrincipalNamesRaw, &ca.CertificateUserPrincipalNames); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshalling certificate user principal names")
		}
	}

	if includeSecrets {
		// Decrypt sensitive fields
		if ca.APITokenEncrypted != nil {
			decryptedAPIToken, err := decrypt(ca.APITokenEncrypted, ds.serverPrivateKey)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "decrypting API token")
			}
			ca.APIToken = ptr.String(string(decryptedAPIToken))
		}
		if ca.PasswordEncrypted != nil {
			decryptedPassword, err := decrypt(ca.PasswordEncrypted, ds.serverPrivateKey)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "decrypting password")
			}
			ca.Password = ptr.String(string(decryptedPassword))
		}
		if ca.ChallengeEncrypted != nil {
			decryptedChallenge, err := decrypt(ca.ChallengeEncrypted, ds.serverPrivateKey)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "decrypting challenge")
			}
			ca.Challenge = ptr.String(string(decryptedChallenge))
		}
		if ca.ClientSecretEncrypted != nil {
			decryptedClientSecret, err := decrypt(ca.ClientSecretEncrypted, ds.serverPrivateKey)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "decrypting client secret")
			}
			ca.ClientSecret = ptr.String(string(decryptedClientSecret))
		}
	} else {
		if ca.APITokenEncrypted != nil {
			ca.APIToken = ptr.String(fleet.MaskedPassword)
		}
		if ca.PasswordEncrypted != nil {
			ca.Password = ptr.String(fleet.MaskedPassword)
		}
		if ca.ChallengeEncrypted != nil {
			ca.Challenge = ptr.String(fleet.MaskedPassword)
		}
		if ca.ClientSecretEncrypted != nil {
			ca.ClientSecret = ptr.String(fleet.MaskedPassword)
		}
	}

	return &ca.CertificateAuthority, nil
}

func (ds *Datastore) GetAllCertificateAuthorities(ctx context.Context, includeSecrets bool) ([]*fleet.CertificateAuthority, error) {
	stmt := `
	SELECT
		id,
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
		FROM
			certificate_authorities
		`

	var cas []certificateAuthorityWithEncryptedSecrets
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cas, stmt); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "select CertificateAuthorities")
	}

	processedCAs := make([]*fleet.CertificateAuthority, 0, len(cas))

	for _, ca := range cas {
		if ca.CertificateUserPrincipalNamesRaw != nil {
			if err := json.Unmarshal(ca.CertificateUserPrincipalNamesRaw, &ca.CertificateUserPrincipalNames); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "unmarshalling certificate user principal names")
			}
		}

		if includeSecrets {
			// Decrypt sensitive fields
			if ca.APITokenEncrypted != nil {
				decryptedAPIToken, err := decrypt(ca.APITokenEncrypted, ds.serverPrivateKey)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting API token for certificate authority %d", ca.ID))
				}
				ca.APIToken = ptr.String(string(decryptedAPIToken))
			}
			if ca.PasswordEncrypted != nil {
				decryptedPassword, err := decrypt(ca.PasswordEncrypted, ds.serverPrivateKey)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting password for certificate authority %d", ca.ID))
				}
				ca.Password = ptr.String(string(decryptedPassword))
			}
			if ca.ChallengeEncrypted != nil {
				decryptedChallenge, err := decrypt(ca.ChallengeEncrypted, ds.serverPrivateKey)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting challenge for certificate authority %d", ca.ID))
				}
				ca.Challenge = ptr.String(string(decryptedChallenge))
			}
			if ca.ClientSecretEncrypted != nil {
				decryptedClientSecret, err := decrypt(ca.ClientSecretEncrypted, ds.serverPrivateKey)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting client secret for certificate authority %d", ca.ID))
				}
				ca.ClientSecret = ptr.String(string(decryptedClientSecret))
			}
		} else {
			if ca.APITokenEncrypted != nil {
				ca.APIToken = ptr.String(fleet.MaskedPassword)
			}
			if ca.PasswordEncrypted != nil {
				ca.Password = ptr.String(fleet.MaskedPassword)
			}
			if ca.ChallengeEncrypted != nil {
				ca.Challenge = ptr.String(fleet.MaskedPassword)
			}
			if ca.ClientSecretEncrypted != nil {
				ca.ClientSecret = ptr.String(fleet.MaskedPassword)
			}
		}
		processedCAs = append(processedCAs, &ca.CertificateAuthority)
	}

	return processedCAs, nil
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

func (ds *Datastore) DeleteCertificateAuthority(ctx context.Context, certificateAuthorityID uint) (*fleet.CertificateAuthoritySummary, error) {
	stmt := `
	SELECT
		id, name, type
	FROM
		certificate_authorities
	WHERE
		id = ?
	`

	var ca fleet.CertificateAuthoritySummary
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &ca, stmt, certificateAuthorityID); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, common_mysql.NotFound(fmt.Sprintf("certificate authority with id %d", certificateAuthorityID))
		}
		return nil, ctxerr.Wrapf(ctx, err, "check certificate authority existence")
	}

	stmt = "DELETE FROM certificate_authorities WHERE id = ?"
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

	return &ca, nil
}
