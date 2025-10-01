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
		challenge_url,
		username,
		password_encrypted,
		challenge_encrypted,
		client_id,
		client_secret_encrypted,
		created_at,
		updated_at
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

	if err := ds.postprocessRetrievedCertificateAuthority(ctx, &ca, includeSecrets); err != nil {
		return nil, err
	}

	return &ca.CertificateAuthority, nil
}

func (ds *Datastore) postprocessRetrievedCertificateAuthority(ctx context.Context, ca *certificateAuthorityWithEncryptedSecrets, includeSecrets bool) error {
	if includeSecrets {
		// Decrypt sensitive fields
		if ca.APITokenEncrypted != nil {
			decryptedAPIToken, err := decrypt(ca.APITokenEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting API token for certificate authority %d", ca.ID))
			}
			ca.APIToken = ptr.String(string(decryptedAPIToken))
		}
		if ca.PasswordEncrypted != nil {
			decryptedPassword, err := decrypt(ca.PasswordEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting password for certificate authority %d", ca.ID))
			}
			ca.Password = ptr.String(string(decryptedPassword))
		}
		if ca.ChallengeEncrypted != nil {
			decryptedChallenge, err := decrypt(ca.ChallengeEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting challenge for certificate authority %d", ca.ID))
			}
			ca.Challenge = ptr.String(string(decryptedChallenge))
		}
		if ca.ClientSecretEncrypted != nil {
			decryptedClientSecret, err := decrypt(ca.ClientSecretEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting client secret for certificate authority %d", ca.ID))
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
	if ca.CertificateUserPrincipalNamesRaw != nil {
		if err := json.Unmarshal(ca.CertificateUserPrincipalNamesRaw, &ca.CertificateUserPrincipalNames); err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshalling certificate user principal names")
		}
	}
	return nil
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
		challenge_url,
		challenge_encrypted,
		client_id,
		client_secret_encrypted,
		created_at,
		updated_at
		FROM
			certificate_authorities
		`

	var cas []certificateAuthorityWithEncryptedSecrets
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cas, stmt); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "select CertificateAuthorities")
	}

	processedCAs := make([]*fleet.CertificateAuthority, 0, len(cas))

	for _, ca := range cas {
		if err := ds.postprocessRetrievedCertificateAuthority(ctx, &ca, includeSecrets); err != nil {
			return nil, err
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

	var cas []*fleet.CertificateAuthoritySummary = []*fleet.CertificateAuthoritySummary{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cas, stmt); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "list CertificateAuthorities")
	}

	return cas, nil
}

func (ds *Datastore) GetGroupedCertificateAuthorities(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
	allCertificateAuthorities, err := ds.GetAllCertificateAuthorities(ctx, includeSecrets)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting all certificates for grouping")
	}

	return fleet.GroupCertificateAuthoritiesByType(allCertificateAuthorities)
}

// Create CA. MUST include secrets
func (ds *Datastore) NewCertificateAuthority(ctx context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
	args, placeholders, err := sqlGenerateArgsForInsertCertificateAuthority(ctx, ds.serverPrivateKey, ca)
	if err != nil {
		return nil, err
	}

	result, err := ds.writer(ctx).ExecContext(ctx, fmt.Sprintf(sqlInsertCertificateAuthority, placeholders), args...)
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

const argsCountInsertCertificateAuthority = 15

const sqlInsertCertificateAuthority = `INSERT INTO certificate_authorities (
	type,
	name,
	url,
	api_token_encrypted,
	profile_id,
	certificate_common_name,
	certificate_user_principal_names,
	certificate_seat_id,
	admin_url,
	challenge_url,
	username,
	password_encrypted,
	challenge_encrypted,
	client_id,
	client_secret_encrypted
) VALUES %s`

const sqlUpsertCertificateAuthority = sqlInsertCertificateAuthority + ` ON DUPLICATE KEY UPDATE
	type = VALUES(type),
	name = VALUES(name),
	url = VALUES(url),
	api_token_encrypted = VALUES(api_token_encrypted),
	profile_id = VALUES(profile_id),
	certificate_common_name = VALUES(certificate_common_name),
	certificate_user_principal_names = VALUES(certificate_user_principal_names),
	certificate_seat_id = VALUES(certificate_seat_id),
	admin_url = VALUES(admin_url),
	challenge_url = VALUES(challenge_url),
	username = VALUES(username),
	password_encrypted = VALUES(password_encrypted),
	challenge_encrypted = VALUES(challenge_encrypted),
	client_id = VALUES(client_id),
	client_secret_encrypted = VALUES(client_secret_encrypted)`

func sqlGenerateArgsForInsertCertificateAuthority(ctx context.Context, serverPrivateKey string, ca *fleet.CertificateAuthority) ([]interface{}, string, error) {
	var upns []byte
	var encryptedPassword []byte
	var encryptedChallenge []byte
	var encryptedAPIToken []byte
	var encryptedClientSecret []byte
	var err error

	if ca.CertificateUserPrincipalNames != nil {
		upns, err = json.Marshal(*ca.CertificateUserPrincipalNames)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "marshalling certificate user principal names for new certificate authority")
		}
	}
	if ca.APIToken != nil {
		encryptedAPIToken, err = encrypt([]byte(*ca.APIToken), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting API token for new certificate authority")
		}
	}
	if ca.Password != nil {
		encryptedPassword, err = encrypt([]byte(*ca.Password), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting password for new certificate authority")
		}
	}
	if ca.Challenge != nil {
		encryptedChallenge, err = encrypt([]byte(*ca.Challenge), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting challenge for new certificate authority")
		}
	}
	if ca.ClientSecret != nil {
		encryptedClientSecret, err = encrypt([]byte(*ca.ClientSecret), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting client secret for new certificate authority")
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
		ca.ChallengeURL,
		ca.Username,
		encryptedPassword,
		encryptedChallenge,
		ca.ClientID,
		encryptedClientSecret,
	}
	placeholders := "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	return args, placeholders, nil
}

func batchUpsertCertificateAuthorities(ctx context.Context, tx sqlx.ExtContext, serverPrivateKey string, certificateAuthorities []*fleet.CertificateAuthority) error {
	if len(certificateAuthorities) == 0 {
		return nil
	}

	var placeholders strings.Builder
	args := make([]interface{}, 0, len(certificateAuthorities)*argsCountInsertCertificateAuthority)

	for _, ca := range certificateAuthorities {
		a, p, err := sqlGenerateArgsForInsertCertificateAuthority(ctx, serverPrivateKey, ca)
		if err != nil {
			return err
		}
		args = append(args, a...)
		placeholders.WriteString(fmt.Sprintf("%s,", p))
	}

	stmt := fmt.Sprintf(sqlUpsertCertificateAuthority, strings.TrimSuffix(placeholders.String(), ","))

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "upserting certificate authorities")
	}

	return nil
}

func batchDeleteCertificateAuthorities(ctx context.Context, tx sqlx.ExtContext, certificateAuthorities []*fleet.CertificateAuthority) error {
	if len(certificateAuthorities) == 0 {
		return nil
	}

	stmt := `DELETE FROM certificate_authorities WHERE (name, type) IN (%s)`
	args := make([]interface{}, 0, len(certificateAuthorities)*2)
	var placeholders strings.Builder
	for _, ca := range certificateAuthorities {
		args = append(args, ca.Name, ca.Type)
		placeholders.WriteString("(?, ?),")
	}
	stmt = fmt.Sprintf(stmt, strings.TrimSuffix(placeholders.String(), ","))

	_, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting certificate authorities")
	}

	return nil
}

func (ds *Datastore) BatchApplyCertificateAuthorities(ctx context.Context, ops fleet.CertificateAuthoritiesBatchOperations,
) error {
	upserts := make([]*fleet.CertificateAuthority, 0, len(ops.Add)+len(ops.Update))
	upserts = append(upserts, ops.Add...)
	upserts = append(upserts, ops.Update...)

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := batchDeleteCertificateAuthorities(ctx, tx, ops.Delete); err != nil {
			return err
		}
		if err := batchUpsertCertificateAuthorities(ctx, tx, ds.serverPrivateKey, upserts); err != nil {
			return err
		}
		return nil
	})
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

func (ds *Datastore) UpdateCertificateAuthorityByID(ctx context.Context, certificateAuthorityID uint, ca *fleet.CertificateAuthority) error {
	oldCA, err := ds.GetCertificateAuthorityByID(ctx, certificateAuthorityID, false)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting certificate authority with id %d", certificateAuthorityID)
	}

	// If the name is being updated, check if it's the same as the old one.
	sameName := ca.Name != nil && *oldCA.Name == *ca.Name
	if sameName {
		return fleet.ConflictError{Message: "a certificate authority with this name already exists"}
	}

	var updateArgs []any

	setStmt, err := ds.generateUpdateQueryWithArgs(ctx, ca, &updateArgs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating update query for certificate authority")
	}
	updateArgs = append(updateArgs, certificateAuthorityID)

	stmt := fmt.Sprintf(`
	UPDATE certificate_authorities
	%s
	WHERE id = ?
	`, setStmt)

	_, err = ds.writer(ctx).ExecContext(ctx, stmt, updateArgs...)
	if err != nil {
		if strings.Contains(err.Error(), "idx_ca_type_name") {
			return fleet.ConflictError{Message: "a certificate authority with this name already exists"}
		}
		return ctxerr.Wrapf(ctx, err, "updating certificate authority with id %d", certificateAuthorityID)
	}

	return nil
}

// generateUpdateQuery generates the SQL update query for a Certificate Authority based on the provided fields
// and will also return the arguments to be used in the query.
func (ds *Datastore) generateUpdateQueryWithArgs(ctx context.Context, ca *fleet.CertificateAuthority, args *[]any) (string, error) {
	updates := []string{}
	switch ca.Type {
	case string(fleet.CATypeDigiCert):
		if ca.Name != nil {
			updates = append(updates, "name = ?")
			*args = append(*args, *ca.Name)
		}
		if ca.URL != nil {
			updates = append(updates, "url = ?")
			*args = append(*args, *ca.URL)
		}
		if ca.APIToken != nil {
			updates = append(updates, "api_token_encrypted = ?")
			encryptedAPIToken, err := encrypt([]byte(*ca.APIToken), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting API token for new certificate authority")
			}
			*args = append(*args, encryptedAPIToken)
		}
		if ca.ProfileID != nil {
			updates = append(updates, "profile_id = ?")
			*args = append(*args, *ca.ProfileID)
		}
		if ca.CertificateCommonName != nil {
			updates = append(updates, "certificate_common_name = ?")
			*args = append(*args, *ca.CertificateCommonName)
		}
		if ca.CertificateUserPrincipalNames != nil {
			updates = append(updates, "certificate_user_principal_names = ?")
			upns, err := json.Marshal(*ca.CertificateUserPrincipalNames)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "marshalling certificate user principal names for updating certificate authority")
			}
			*args = append(*args, upns)
		}
		if ca.CertificateSeatID != nil {
			updates = append(updates, "certificate_seat_id = ?")
			*args = append(*args, *ca.CertificateSeatID)
		}
	case string(fleet.CATypeHydrant):
		if ca.URL != nil {
			updates = append(updates, "url = ?")
			*args = append(*args, *ca.URL)
		}
		if ca.Name != nil {
			updates = append(updates, "name = ?")
			*args = append(*args, *ca.Name)
		}
		if ca.ClientID != nil {
			updates = append(updates, "client_id = ?")
			*args = append(*args, *ca.ClientID)
		}
		if ca.ClientSecret != nil {
			updates = append(updates, "client_secret_encrypted = ?")
			encryptedClientSecret, err := encrypt([]byte(*ca.ClientSecret), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting client secret for new certificate authority")
			}
			*args = append(*args, encryptedClientSecret)
		}

	case string(fleet.CATypeNDESSCEPProxy):
		if ca.URL != nil {
			updates = append(updates, "url = ?")
			*args = append(*args, *ca.URL)
		}
		if ca.AdminURL != nil {
			updates = append(updates, "admin_url = ?")
			*args = append(*args, *ca.AdminURL)
		}
		if ca.Username != nil {
			updates = append(updates, "username = ?")
			*args = append(*args, *ca.Username)
		}
		if ca.Password != nil {
			updates = append(updates, "password_encrypted = ?")
			encryptedPassword, err := encrypt([]byte(*ca.Password), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting password for new certificate authority")
			}
			*args = append(*args, encryptedPassword)
		}

	case string(fleet.CATypeCustomSCEPProxy):
		if ca.Name != nil {
			updates = append(updates, "name = ?")
			*args = append(*args, *ca.Name)
		}
		if ca.URL != nil {
			updates = append(updates, "url = ?")
			*args = append(*args, *ca.URL)
		}
		if ca.Challenge != nil {
			updates = append(updates, "challenge_encrypted = ?")
			encryptedChallenge, err := encrypt([]byte(*ca.Challenge), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting challenge for new certificate authority")
			}
			*args = append(*args, encryptedChallenge)
		}
	case string(fleet.CATypeSmallstep):
		if ca.Name != nil {
			updates = append(updates, "name = ?")
			*args = append(*args, *ca.Name)
		}
		if ca.URL != nil {
			updates = append(updates, "url = ?")
			*args = append(*args, *ca.URL)
		}
		if ca.ChallengeURL != nil {
			updates = append(updates, "challenge_url = ?")
			*args = append(*args, *ca.ChallengeURL)
		}
		if ca.Username != nil {
			updates = append(updates, "username = ?")
			*args = append(*args, *ca.Username)
		}
		if ca.Password != nil {
			updates = append(updates, "password_encrypted = ?")
			encryptedPassword, err := encrypt([]byte(*ca.Password), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting password for new certificate authority")
			}
			*args = append(*args, encryptedPassword)
		}
	default:
		return "", fmt.Errorf("unknown certificate authority type: %s", ca.Type)
	}
	return fmt.Sprintf("SET %s", strings.Join(updates, ", ")), nil
}
