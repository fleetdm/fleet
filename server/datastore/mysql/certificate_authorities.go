package mysql

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/crypto"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

type certificateAuthorityWithEncryptedSecrets struct {
	fleet.CertificateAuthority
	APITokenEncrypted                []byte `db:"api_token_encrypted"`
	PasswordEncrypted                []byte `db:"password_encrypted"`
	ChallengeEncrypted               []byte `db:"challenge_encrypted"`
	ClientSecretEncrypted            []byte `db:"client_secret_encrypted"`
	MTLSKeyEncrypted                 []byte `db:"mtls_key_encrypted"`
	GoogleCASAPIKeyJSONEncrypted     []byte `db:"google_cas_api_key_json_encrypted"`
	CertificateUserPrincipalNamesRaw []byte `db:"certificate_user_principal_names"`
}

// ejbcaColumns are the EJBCA columns of certificate_authorities, selected after client_secret_encrypted.
const ejbcaColumns = `
		mtls_cert_pem,
		mtls_key_encrypted,
		trust_ca_bundle_pem,
		ejbca_ca_name,
		ejbca_certificate_profile_name,
		ejbca_end_entity_profile_name,
		ejbca_username_template,`

// googleCASColumns are the Google CAS columns of certificate_authorities, selected after the EJBCA ones.
const googleCASColumns = `
		google_cas_project,
		google_cas_location,
		google_cas_ca_pool,
		google_cas_api_key_json_encrypted,`

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
		client_secret_encrypted,` + ejbcaColumns + googleCASColumns + `
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
			decryptedAPIToken, err := crypto.DecryptAESGCM(ca.APITokenEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting API token for certificate authority %d", ca.ID))
			}
			ca.APIToken = ptr.String(string(decryptedAPIToken))
		}
		if ca.PasswordEncrypted != nil {
			decryptedPassword, err := crypto.DecryptAESGCM(ca.PasswordEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting password for certificate authority %d", ca.ID))
			}
			ca.Password = ptr.String(string(decryptedPassword))
		}
		if ca.ChallengeEncrypted != nil {
			decryptedChallenge, err := crypto.DecryptAESGCM(ca.ChallengeEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting challenge for certificate authority %d", ca.ID))
			}
			ca.Challenge = ptr.String(string(decryptedChallenge))
		}
		if ca.ClientSecretEncrypted != nil {
			decryptedClientSecret, err := crypto.DecryptAESGCM(ca.ClientSecretEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting client secret for certificate authority %d", ca.ID))
			}
			ca.ClientSecret = ptr.String(string(decryptedClientSecret))
		}
		if ca.MTLSKeyEncrypted != nil {
			decryptedMTLSKey, err := crypto.DecryptAESGCM(ca.MTLSKeyEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting mTLS key for certificate authority %d", ca.ID))
			}
			ca.MTLSKeyPEM = new(string(decryptedMTLSKey))
		}
		if ca.GoogleCASAPIKeyJSONEncrypted != nil {
			decryptedAPIKeyJSON, err := crypto.DecryptAESGCM(ca.GoogleCASAPIKeyJSONEncrypted, ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("decrypting Google CAS service account key for certificate authority %d", ca.ID))
			}
			ca.GoogleCASAPIKeyJSON = new(string(decryptedAPIKeyJSON))
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
		if ca.MTLSKeyEncrypted != nil {
			ca.MTLSKeyPEM = new(fleet.MaskedPassword)
		}
		if ca.GoogleCASAPIKeyJSONEncrypted != nil {
			ca.GoogleCASAPIKeyJSON = new(fleet.MaskedPassword)
		}
	}
	if ca.MTLSCertPEM != nil {
		ca.MTLSCertExpiresAt = ds.mtlsCertExpiry(ctx, ca.ID, *ca.MTLSCertPEM)
	}
	if ca.CertificateUserPrincipalNamesRaw != nil {
		if err := json.Unmarshal(ca.CertificateUserPrincipalNamesRaw, &ca.CertificateUserPrincipalNames); err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshalling certificate user principal names")
		}
	}
	return nil
}

// mtlsCertExpiry returns the expiry of the first certificate in the PEM, like the TLS client does, or nil, logged, if it can't be
// parsed so reads don't break.
func (ds *Datastore) mtlsCertExpiry(ctx context.Context, caID uint, certPEM string) *time.Time {
	cert, err := certificate.FirstCertificateFromPEM([]byte(certPEM))
	if err != nil {
		ds.logger.DebugContext(ctx, "parsing certificate authority mTLS certificate", "certificate_authority_id", caID, "err", err)
		return nil
	}
	if cert == nil {
		ds.logger.DebugContext(ctx, "certificate authority mTLS certificate has no PEM certificate", "certificate_authority_id", caID)
		return nil
	}
	return &cert.NotAfter
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
		client_secret_encrypted,` + ejbcaColumns + googleCASColumns + `
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
		id, name, type, mtls_cert_pem
	FROM
		certificate_authorities
	ORDER BY
		name
	`

	var cas []*fleet.CertificateAuthoritySummary = []*fleet.CertificateAuthoritySummary{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cas, stmt); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "list CertificateAuthorities")
	}
	for _, ca := range cas {
		if ca.MTLSCertPEM != nil {
			ca.MTLSCertExpiresAt = ds.mtlsCertExpiry(ctx, ca.ID, *ca.MTLSCertPEM)
			ca.MTLSCertPEM = nil
		}
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

	var id int64
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		result, err := tx.ExecContext(ctx, fmt.Sprintf(sqlInsertCertificateAuthority, placeholders), args...)
		if err != nil {
			if strings.Contains(err.Error(), "idx_ca_type_name") {
				return fleet.ConflictError{Message: "a certificate authority with this name already exists"}
			}
			return ctxerr.Wrap(ctx, err, "inserting new certificate authority")
		}
		id, err = result.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting last insert ID for new certificate authority")
		}
		return ensureEJBCANamesNotSharedDB(ctx, tx, *ca.Name)
	})
	if err != nil {
		return nil, err
	}
	ca.ID = uint(id) //nolint:gosec // dismiss G115
	return ca, nil
}

// ensureEJBCANamesNotSharedDB fails if one of the names is used by an EJBCA certificate authority and by another certificate authority.
func ensureEJBCANamesNotSharedDB(ctx context.Context, tx sqlx.ExtContext, names ...string) error {
	// FOR UPDATE is needed to keep 2 concurrent writes from both passing the check
	stmt, args, err := sqlx.In(`SELECT name, type FROM certificate_authorities WHERE name IN (?) FOR UPDATE`, names)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare check of EJBCA certificate authority names")
	}
	var rows []struct {
		Name string `db:"name"`
		Type string `db:"type"`
	}
	if err := sqlx.SelectContext(ctx, tx, &rows, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "check EJBCA certificate authority names")
	}
	// The names are compared without regard to case, like the DB column is.
	ejbca, other := map[string]string{}, map[string]struct{}{}
	for _, row := range rows {
		if row.Type == string(fleet.CATypeEJBCA) {
			ejbca[strings.ToLower(row.Name)] = row.Name
		} else {
			other[strings.ToLower(row.Name)] = struct{}{}
		}
	}
	for lowerName, name := range ejbca {
		if _, shared := other[lowerName]; shared {
			return fleet.EJBCANameInUseError{Name: name}
		}
	}
	return nil
}

const argsCountInsertCertificateAuthority = 26

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
	client_secret_encrypted,
	mtls_cert_pem,
	mtls_key_encrypted,
	trust_ca_bundle_pem,
	ejbca_ca_name,
	ejbca_certificate_profile_name,
	ejbca_end_entity_profile_name,
	ejbca_username_template,
	google_cas_project,
	google_cas_location,
	google_cas_ca_pool,
	google_cas_api_key_json_encrypted
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
	client_secret_encrypted = VALUES(client_secret_encrypted),
	mtls_cert_pem = VALUES(mtls_cert_pem),
	mtls_key_encrypted = VALUES(mtls_key_encrypted),
	trust_ca_bundle_pem = VALUES(trust_ca_bundle_pem),
	ejbca_ca_name = VALUES(ejbca_ca_name),
	ejbca_certificate_profile_name = VALUES(ejbca_certificate_profile_name),
	ejbca_end_entity_profile_name = VALUES(ejbca_end_entity_profile_name),
	ejbca_username_template = VALUES(ejbca_username_template),
	google_cas_project = VALUES(google_cas_project),
	google_cas_location = VALUES(google_cas_location),
	google_cas_ca_pool = VALUES(google_cas_ca_pool),
	google_cas_api_key_json_encrypted = VALUES(google_cas_api_key_json_encrypted)`

func sqlGenerateArgsForInsertCertificateAuthority(ctx context.Context, serverPrivateKey string, ca *fleet.CertificateAuthority) ([]interface{}, string, error) {
	var upns []byte
	var encryptedPassword []byte
	var encryptedChallenge []byte
	var encryptedAPIToken []byte
	var encryptedClientSecret []byte
	var encryptedMTLSKey []byte
	var encryptedGoogleCASAPIKeyJSON []byte
	var err error

	if ca.CertificateUserPrincipalNames != nil {
		upns, err = json.Marshal(*ca.CertificateUserPrincipalNames)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "marshalling certificate user principal names for new certificate authority")
		}
	}
	if ca.APIToken != nil {
		encryptedAPIToken, err = crypto.EncryptAESGCM([]byte(*ca.APIToken), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting API token for new certificate authority")
		}
	}
	if ca.Password != nil {
		encryptedPassword, err = crypto.EncryptAESGCM([]byte(*ca.Password), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting password for new certificate authority")
		}
	}
	if ca.Challenge != nil {
		encryptedChallenge, err = crypto.EncryptAESGCM([]byte(*ca.Challenge), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting challenge for new certificate authority")
		}
	}
	if ca.ClientSecret != nil {
		encryptedClientSecret, err = crypto.EncryptAESGCM([]byte(*ca.ClientSecret), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting client secret for new certificate authority")
		}
	}
	if ca.MTLSKeyPEM != nil {
		encryptedMTLSKey, err = crypto.EncryptAESGCM([]byte(*ca.MTLSKeyPEM), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting mTLS key for new certificate authority")
		}
	}
	if ca.GoogleCASAPIKeyJSON != nil {
		encryptedGoogleCASAPIKeyJSON, err = crypto.EncryptAESGCM([]byte(*ca.GoogleCASAPIKeyJSON), serverPrivateKey)
		if err != nil {
			return nil, "", ctxerr.Wrap(ctx, err, "encrypting Google CAS service account key for new certificate authority")
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
		ca.MTLSCertPEM,
		encryptedMTLSKey,
		ca.TrustCABundlePEM,
		ca.EJBCACAName,
		ca.EJBCACertificateProfileName,
		ca.EJBCAEndEntityProfileName,
		ca.EJBCAUsernameTemplate,
		ca.GoogleCASProject,
		ca.GoogleCASLocation,
		ca.GoogleCASCAPool,
		encryptedGoogleCASAPIKeyJSON,
	}
	placeholders := "(" + strings.TrimSuffix(strings.Repeat("?,", len(args)), ",") + ")"

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
		if isMySQLForeignKey(err) {
			return &fleet.ConflictError{
				Message: "Couldn't delete certificate authority. " + fleet.DeleteCAReferencedByTemplatesErrMsg + ". Please remove the certificate templates first.",
			}
		}
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
		resendCANames := make(map[fleet.CAType][]string, len(profileCATypes))
		for _, t := range profileCATypes {
			var updates []*fleet.CertificateAuthority
			var names []string
			for _, ca := range ops.Update {
				if ca.Type == string(t.caType) {
					updates = append(updates, ca)
					names = append(names, *ca.Name)
				}
			}
			if len(updates) == 0 {
				continue
			}
			oldCAs, err := ds.getCAsForUpdateDB(ctx, tx, t.caType, "name IN (?)", names)
			if err != nil {
				return err
			}
			for _, ca := range updates {
				if old, ok := oldCAs[strings.ToLower(*ca.Name)]; ok && certificateFieldsChanged(old, ca, true) {
					resendCANames[t.caType] = append(resendCANames[t.caType], *ca.Name)
				}
			}
		}
		if err := batchUpsertCertificateAuthorities(ctx, tx, ds.serverPrivateKey, upserts); err != nil {
			return err
		}
		if err := batchDeleteCertificateAuthorities(ctx, tx, ops.Delete); err != nil {
			return err
		}
		// Checked after the deletes, so a CA that is replaced by one of another type with the same name, as a rename is in GitOps, is fine.
		if len(upserts) > 0 {
			names := make([]string, 0, len(upserts))
			for _, ca := range upserts {
				names = append(names, *ca.Name)
			}
			if err := ensureEJBCANamesNotSharedDB(ctx, tx, names...); err != nil {
				return err
			}
		}
		for _, t := range profileCATypes {
			if names := resendCANames[t.caType]; len(names) > 0 {
				if err := resendCAProfilesDB(ctx, tx, t.dataVar, names, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// profileCATypes are the CA types that issue a certificate for each host when a profile that uses their data variable is delivered.
var profileCATypes = []struct {
	caType  fleet.CAType
	dataVar fleet.FleetVarName
}{
	{fleet.CATypeDigiCert, fleet.FleetVarDigiCertDataPrefix},
	{fleet.CATypeEJBCA, fleet.FleetVarEJBCADataPrefix},
}

// getCAsForUpdateDB locks the CAs of a type matching where, selecting only the fields that certificateFieldsChanged compares. The result is
// keyed by the lowercase name, because the column compares names without regard to case.
func (ds *Datastore) getCAsForUpdateDB(ctx context.Context, tx sqlx.ExtContext, caType fleet.CAType, where string, args ...any,
) (map[string]*fleet.CertificateAuthority, error) {
	stmt, args, err := sqlx.In(`
		SELECT
			id, type, name, profile_id, certificate_common_name, certificate_user_principal_names, certificate_seat_id, ejbca_ca_name,
			ejbca_certificate_profile_name, ejbca_end_entity_profile_name, ejbca_username_template
		FROM certificate_authorities
		WHERE type = ? AND `+where+`
		FOR UPDATE`, append([]any{caType}, args...)...)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "prepare select %s certificate authorities for update", caType)
	}
	var rows []certificateAuthorityWithEncryptedSecrets
	if err := sqlx.SelectContext(ctx, tx, &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "select %s certificate authorities for update", caType)
	}
	cas := make(map[string]*fleet.CertificateAuthority, len(rows))
	for _, row := range rows {
		if err := ds.postprocessRetrievedCertificateAuthority(ctx, &row, false); err != nil {
			return nil, err
		}
		cas[strings.ToLower(*row.Name)] = &row.CertificateAuthority
	}
	return cas, nil
}

// certificateFieldsChanged reports whether an update changes what the CA puts in issued certificates. Nil fields are unchanged in a
// partial update, but cleared in a full replacement such as a batch apply.
func certificateFieldsChanged(oldCA, ca *fleet.CertificateAuthority, fullReplacement bool) bool {
	upnsChanged := (ca.CertificateUserPrincipalNames != nil || fullReplacement) &&
		!slices.Equal(ptr.ValOrZero(ca.CertificateUserPrincipalNames), ptr.ValOrZero(oldCA.CertificateUserPrincipalNames))
	changed := func(updated, old *string) bool {
		return (updated != nil || fullReplacement) && ptr.ValOrZero(updated) != ptr.ValOrZero(old)
	}
	switch fleet.CAType(ca.Type) {
	case fleet.CATypeDigiCert:
		return changed(ca.ProfileID, oldCA.ProfileID) || changed(ca.CertificateCommonName, oldCA.CertificateCommonName) ||
			changed(ca.CertificateSeatID, oldCA.CertificateSeatID) || upnsChanged
	case fleet.CATypeEJBCA:
		return changed(ca.EJBCACAName, oldCA.EJBCACAName) || changed(ca.EJBCACertificateProfileName, oldCA.EJBCACertificateProfileName) ||
			changed(ca.EJBCAEndEntityProfileName, oldCA.EJBCAEndEntityProfileName) ||
			changed(ca.CertificateCommonName, oldCA.CertificateCommonName) ||
			changed(ca.EJBCAUsernameTemplate, oldCA.EJBCAUsernameTemplate) || upnsChanged
	default:
		return false
	}
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
		if isMySQLForeignKey(err) {
			return nil, fleet.ConflictError{
				Message: "Couldn't delete certificate authority. " + fleet.DeleteCAReferencedByTemplatesErrMsg + ". Please remove the certificate templates first.",
			}
		}
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

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var resendCAName *string
		dataVar, resends := dataVarForCAType(fleet.CAType(ca.Type))
		if resends {
			oldCAs, err := ds.getCAsForUpdateDB(ctx, tx, fleet.CAType(ca.Type), "id = ?", certificateAuthorityID)
			if err != nil {
				return err
			}
			for _, old := range oldCAs {
				if certificateFieldsChanged(old, ca, false) {
					resendCAName = cmp.Or(ca.Name, old.Name)
				}
			}
		}

		if _, err := tx.ExecContext(ctx, stmt, updateArgs...); err != nil {
			if strings.Contains(err.Error(), "idx_ca_type_name") {
				return fleet.ConflictError{Message: "a certificate authority with this name already exists"}
			}
			return ctxerr.Wrapf(ctx, err, "updating certificate authority with id %d", certificateAuthorityID)
		}
		if ca.Name != nil {
			if err := ensureEJBCANamesNotSharedDB(ctx, tx, *ca.Name); err != nil {
				return err
			}
		}

		if resendCAName == nil {
			return nil
		}
		return resendCAProfilesDB(ctx, tx, dataVar, []string{*resendCAName}, nil)
	})
}

// dataVarForCAType returns the prefix of the profile variable that gets a host its certificate from a CA of the type, if it has one.
func dataVarForCAType(caType fleet.CAType) (fleet.FleetVarName, bool) {
	for _, t := range profileCATypes {
		if t.caType == caType {
			return t.dataVar, true
		}
	}
	return "", false
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
			encryptedAPIToken, err := crypto.EncryptAESGCM([]byte(*ca.APIToken), ds.serverPrivateKey)
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
			encryptedClientSecret, err := crypto.EncryptAESGCM([]byte(*ca.ClientSecret), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting client secret for new certificate authority")
			}
			*args = append(*args, encryptedClientSecret)
		}

	case string(fleet.CATypeCustomESTProxy):
		if ca.URL != nil {
			updates = append(updates, "url = ?")
			*args = append(*args, *ca.URL)
		}
		if ca.Name != nil {
			updates = append(updates, "name = ?")
			*args = append(*args, *ca.Name)
		}
		if ca.Username != nil {
			updates = append(updates, "username = ?")
			*args = append(*args, *ca.Username)
		}
		if ca.Password != nil {
			updates = append(updates, "password_encrypted = ?")
			encryptedPassword, err := crypto.EncryptAESGCM([]byte(*ca.Password), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting password for new certificate authority")
			}
			*args = append(*args, encryptedPassword)
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
			encryptedPassword, err := crypto.EncryptAESGCM([]byte(*ca.Password), ds.serverPrivateKey)
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
			encryptedChallenge, err := crypto.EncryptAESGCM([]byte(*ca.Challenge), ds.serverPrivateKey)
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
			encryptedPassword, err := crypto.EncryptAESGCM([]byte(*ca.Password), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting password for new certificate authority")
			}
			*args = append(*args, encryptedPassword)
		}
	case string(fleet.CATypeEJBCA):
		for _, f := range []struct {
			column string
			value  *string
			// optional fields are stored as NULL when emptied, as they are when they are not set on create
			optional bool
		}{
			{"name", ca.Name, false},
			{"url", ca.URL, false},
			{"certificate_common_name", ca.CertificateCommonName, false},
			{"mtls_cert_pem", ca.MTLSCertPEM, false},
			{"trust_ca_bundle_pem", ca.TrustCABundlePEM, true},
			{"ejbca_ca_name", ca.EJBCACAName, false},
			{"ejbca_certificate_profile_name", ca.EJBCACertificateProfileName, false},
			{"ejbca_end_entity_profile_name", ca.EJBCAEndEntityProfileName, false},
			{"ejbca_username_template", ca.EJBCAUsernameTemplate, true},
		} {
			if f.value == nil {
				continue
			}
			if f.optional {
				updates = append(updates, f.column+" = NULLIF(?, '')")
			} else {
				updates = append(updates, f.column+" = ?")
			}
			*args = append(*args, *f.value)
		}
		if ca.MTLSKeyPEM != nil {
			updates = append(updates, "mtls_key_encrypted = ?")
			encryptedMTLSKey, err := crypto.EncryptAESGCM([]byte(*ca.MTLSKeyPEM), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting mTLS key for certificate authority")
			}
			*args = append(*args, encryptedMTLSKey)
		}
		if ca.CertificateUserPrincipalNames != nil {
			if len(*ca.CertificateUserPrincipalNames) == 0 {
				updates = append(updates, "certificate_user_principal_names = NULL")
			} else {
				updates = append(updates, "certificate_user_principal_names = ?")
				upns, err := json.Marshal(*ca.CertificateUserPrincipalNames)
				if err != nil {
					return "", ctxerr.Wrap(ctx, err, "marshalling certificate user principal names for updating certificate authority")
				}
				*args = append(*args, upns)
			}
		}
	case string(fleet.CATypeGoogleCAS):
		for _, f := range []struct {
			column string
			value  *string
		}{
			{"name", ca.Name},
			{"google_cas_project", ca.GoogleCASProject},
			{"google_cas_location", ca.GoogleCASLocation},
			{"google_cas_ca_pool", ca.GoogleCASCAPool},
		} {
			if f.value != nil {
				updates = append(updates, f.column+" = ?")
				*args = append(*args, *f.value)
			}
		}
		if ca.GoogleCASAPIKeyJSON != nil {
			updates = append(updates, "google_cas_api_key_json_encrypted = ?")
			encryptedAPIKeyJSON, err := crypto.EncryptAESGCM([]byte(*ca.GoogleCASAPIKeyJSON), ds.serverPrivateKey)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "encrypting Google CAS service account key for certificate authority")
			}
			*args = append(*args, encryptedAPIKeyJSON)
		}
	default:
		return "", fmt.Errorf("unknown certificate authority type: %s", ca.Type)
	}
	return fmt.Sprintf("SET %s", strings.Join(updates, ", ")), nil
}
