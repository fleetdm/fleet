package mysql

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/text/unicode/norm"
)

// secretVariableAllowedOrderKeys defines the allowed order keys for ListSecretVariables.
// SECURITY: This prevents information disclosure via arbitrary column sorting.
// Sensitive columns like 'value' are intentionally excluded.
var secretVariableAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"name":       "name",
	"id":         "id",
	"updated_at": "updated_at",
}

func (ds *Datastore) UpsertSecretVariables(ctx context.Context, secretVariables []fleet.SecretVariable) error {
	if len(secretVariables) == 0 {
		return nil
	}

	// The secret variables should rarely change, so we do not use a transaction here.
	// When we encrypt a secret variable, it is salted, so the encrypted data is different each time.
	// In order to keep the updated_at timestamp correct, we need to compare the encrypted value
	// with the existing value in the database. If the values are the same, we do not update the row.

	var names []string
	for _, secretVariable := range secretVariables {
		names = append(names, secretVariable.Name)
	}
	existingVariables, err := ds.GetSecretVariables(ctx, names)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get existing secret variables")
	}
	existingVariableMap := make(map[string]string, len(existingVariables))
	for _, existingVariable := range existingVariables {
		existingVariableMap[existingVariable.Name] = existingVariable.Value
	}
	var variablesToInsert []fleet.SecretVariable
	var variablesToUpdate []fleet.SecretVariable
	for _, secretVariable := range secretVariables {
		existingValue, ok := existingVariableMap[secretVariable.Name]
		switch {
		case !ok:
			variablesToInsert = append(variablesToInsert, secretVariable)
		case existingValue != secretVariable.Value:
			variablesToUpdate = append(variablesToUpdate, secretVariable)
		default:
			// No change -- the variable value is the same
		}
	}

	if len(variablesToInsert) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(variablesToInsert)), ",")
		stmt := fmt.Sprintf(`
		INSERT INTO secret_variables (name, value)
		VALUES %s`, values)
		args := make([]interface{}, 0, len(variablesToInsert)*2)
		for _, secretVariable := range variablesToInsert {
			valueEncrypted, err := encrypt([]byte(secretVariable.Value), ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "encrypt secret value for insert with server private key")
			}
			args = append(args, secretVariable.Name, valueEncrypted)
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert secret variables")
		}
	}

	if len(variablesToUpdate) > 0 {
		stmt := `
		UPDATE secret_variables
		SET value = ?
		WHERE name = ?`
		for _, secretVariable := range variablesToUpdate {
			valueEncrypted, err := encrypt([]byte(secretVariable.Value), ds.serverPrivateKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "encrypt secret value for update with server private key")
			}
			if _, err := ds.writer(ctx).ExecContext(ctx, stmt, valueEncrypted, secretVariable.Name); err != nil {
				return ctxerr.Wrap(ctx, err, "update secret variables")
			}
		}
	}

	return nil
}

func (ds *Datastore) CreateSecretVariable(ctx context.Context, name string, value string) (id uint, err error) {
	valueEncrypted, err := encrypt([]byte(value), ds.serverPrivateKey)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "encrypt secret value for insert with server private key")
	}
	res, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO secret_variables (name, value) VALUES (?, ?)`,
		name, valueEncrypted,
	)
	if err != nil {
		if IsDuplicate(err) {
			return 0, ctxerr.Wrap(ctx, alreadyExists("name", name), "found duplicate")
		}
		return 0, ctxerr.Wrap(ctx, err, "insert secret variable")
	}
	id_, _ := res.LastInsertId()
	return uint(id_), nil //nolint:gosec // dismiss G115
}

func (ds *Datastore) GetSecretVariables(ctx context.Context, names []string) ([]fleet.SecretVariable, error) {
	if len(names) == 0 {
		return nil, nil
	}

	stmt, args, err := sqlx.In(`
		SELECT name, value, updated_at
		FROM secret_variables
		WHERE name IN (?)`, names)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build secret variables query")
	}

	var secretVariables []fleet.SecretVariable

	err = sqlx.SelectContext(ctx, ds.reader(ctx), &secretVariables, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get secret variables")
	}

	for i, secretVariable := range secretVariables {
		valueDecrypted, err := decrypt([]byte(secretVariable.Value), ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decrypt secret value with server private key")
		}
		secretVariables[i].Value = string(valueDecrypted)
	}

	return secretVariables, nil
}

func (ds *Datastore) ListSecretVariables(ctx context.Context, opt fleet.ListOptions) (
	secretVariables []fleet.SecretVariableIdentifier, meta *fleet.PaginationMetadata, count int, err error,
) {
	stmt := `SELECT id, name, updated_at FROM secret_variables WHERE true`

	// normalize the name for full Unicode support (Unicode equivalence).
	normMatch := norm.NFC.String(opt.MatchQuery)
	whereClauses, args := searchLike("", nil, normMatch, "name")
	stmt += whereClauses

	// perform a second query to grab the count
	// build the count statement before adding pagination constraints
	countStmt := fmt.Sprintf("SELECT COUNT(DISTINCT id) FROM (%s) AS s", stmt)

	stmt, args, err = appendListOptionsWithCursorToSQLSecure(stmt, args, &opt, secretVariableAllowedOrderKeys)
	if err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "apply list options")
	}

	dbReader := ds.reader(ctx)
	if err := sqlx.SelectContext(ctx, dbReader, &secretVariables, stmt, args...); err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "listing secret variables")
	}
	if err := sqlx.GetContext(ctx, dbReader, &count, countStmt, args...); err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "get secret variables count")
	}

	if opt.IncludeMetadata {
		meta = &fleet.PaginationMetadata{
			HasPreviousResults: opt.Page > 0,
			TotalResults:       uint(count), //nolint:gosec // dismiss G115
		}
		// `appendListOptionsWithCursorToSQL` used above to build the query statement will cause this discrepancy.
		if len(secretVariables) > int(opt.PerPage) { //nolint:gosec // dismiss G115
			meta.HasNextResults = true
			secretVariables = secretVariables[:len(secretVariables)-1]
		}
	}

	return secretVariables, meta, count, nil
}

func (ds *Datastore) DeleteSecretVariable(ctx context.Context, id uint) (secretName string, err error) {
	type entity struct {
		// Type is the entity type, "script", "apple_profile", "apple_declaration", or "windows_profile".
		Type string `db:"entity"`
		// Name is the name of the entity.
		Name string `db:"name"`
		// TeamName is the name of the team the entity belongs to.
		TeamName string `db:"team_name"`
		// Contents is the content of the entity (script's/profile's body).
		Contents string `db:"contents"`
	}

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, tx, &secretName, `SELECT name FROM secret_variables WHERE id = ?`, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("SecretVariable").WithID(id))
			}
			return ctxerr.Wrap(ctx, err, "getting name of secret variable to delete")
		}

		// 1. Check if the secret variable is used in scripts.
		var scriptContents []entity
		if err := sqlx.SelectContext(ctx, tx,
			&scriptContents,
			`SELECT 'script' AS entity, s.name,
			COALESCE(t.name, 'No team') AS team_name, sc.contents
			FROM script_contents sc
			JOIN scripts s ON s.script_content_id = sc.id
			LEFT JOIN teams t ON t.id = s.team_id;`,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "get script contents")
		}
		for _, c := range scriptContents {
			if fleet.ContainsVar(c.Contents, fleet.ServerSecretPrefix+secretName) {
				return ctxerr.Wrap(ctx, &fleet.SecretUsedError{
					SecretName: secretName,
					Entity: fleet.EntityUsingSecret{
						Type:     c.Type,
						Name:     c.Name,
						TeamName: c.TeamName,
					},
				}, "found secret in use")
			}
		}

		// 2. Check if the secret variable is used in Apple configuration profiles.
		var appleConfigurationProfileContents []entity
		if err := sqlx.SelectContext(ctx, tx,
			&appleConfigurationProfileContents,
			`SELECT 'apple_profile' AS entity, p.name,
			COALESCE(t.name, 'No team') AS team_name, p.mobileconfig AS contents
			FROM mdm_apple_configuration_profiles p
			LEFT JOIN teams t ON t.id = p.team_id;`,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "get apple profile contents")
		}
		for _, c := range appleConfigurationProfileContents {
			if fleet.ContainsVar(c.Contents, fleet.ServerSecretPrefix+secretName) {
				return ctxerr.Wrap(ctx, &fleet.SecretUsedError{
					SecretName: secretName,
					Entity: fleet.EntityUsingSecret{
						Type:     c.Type,
						Name:     c.Name,
						TeamName: c.TeamName,
					},
				}, "found secret in use")
			}
		}

		// 3. Check if the secret variable is used in Apple declarations.
		var appleDeclarationContents []entity
		if err := sqlx.SelectContext(ctx, tx,
			&appleDeclarationContents,
			`SELECT 'apple_declaration' AS entity, d.name,
			COALESCE(t.name, 'No team') AS team_name, d.raw_json AS contents
			FROM mdm_apple_declarations d
			LEFT JOIN teams t ON t.id = d.team_id;`,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "get apple declaration contents")
		}
		for _, c := range appleDeclarationContents {
			if fleet.ContainsVar(c.Contents, fleet.ServerSecretPrefix+secretName) {
				return ctxerr.Wrap(ctx, &fleet.SecretUsedError{
					SecretName: secretName,
					Entity: fleet.EntityUsingSecret{
						Type:     c.Type,
						Name:     c.Name,
						TeamName: c.TeamName,
					},
				}, "found secret in use")
			}
		}

		// 4. Check if the secret variable is used in Windows configuration profiles.
		var windowsProfileContents []entity
		if err := sqlx.SelectContext(ctx, tx,
			&windowsProfileContents,
			`SELECT 'windows_profile' AS entity, p.name,
			COALESCE(t.name, 'No team') AS team_name, p.syncml AS contents
			FROM mdm_windows_configuration_profiles p
			LEFT JOIN teams t ON t.id = p.team_id;`,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "get windows profile contents")
		}
		for _, c := range windowsProfileContents {
			if fleet.ContainsVar(c.Contents, fleet.ServerSecretPrefix+secretName) {
				return ctxerr.Wrap(ctx, &fleet.SecretUsedError{
					SecretName: secretName,
					Entity: fleet.EntityUsingSecret{
						Type:     c.Type,
						Name:     c.Name,
						TeamName: c.TeamName,
					},
				}, "found secret in use")
			}
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM secret_variables WHERE id = ?`, id); err != nil {
			return ctxerr.Wrap(ctx, err, "delete secret variable")
		}

		return nil
	}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "delete secret variable")
	}

	return secretName, nil
}

func (ds *Datastore) ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error) {
	expanded, _, err := ds.expandEmbeddedSecrets(ctx, document)
	return expanded, err
}

func (ds *Datastore) expandEmbeddedSecrets(ctx context.Context, document string) (string, []fleet.SecretVariable, error) {
	embeddedSecrets := fleet.ContainsPrefixVars(document, fleet.ServerSecretPrefix)
	if len(embeddedSecrets) == 0 {
		return document, nil, nil
	}

	secrets, err := ds.GetSecretVariables(ctx, embeddedSecrets)
	if err != nil {
		return "", nil, ctxerr.Wrap(ctx, err, "expanding embedded secrets")
	}

	secretMap := make(map[string]string, len(secrets))

	for _, secret := range secrets {
		secretMap[secret.Name] = secret.Value
	}

	var missingSecrets []string

	for _, wantSecret := range embeddedSecrets {
		if _, ok := secretMap[wantSecret]; !ok {
			missingSecrets = append(missingSecrets, wantSecret)
		}
	}

	if len(missingSecrets) > 0 {
		return "", nil, fleet.MissingSecretsError{MissingSecrets: missingSecrets}
	}

	// Detect document format so we can escape the secret value appropriately.
	// XML detection is aggressive because Windows profiles do not begin with <?xml.
	trimmed := strings.TrimSpace(document)
	documentIsXML := strings.HasPrefix(trimmed, "<")
	documentIsJSON := strings.HasPrefix(trimmed, "{")

	expanded := fleet.MaybeExpand(document, func(s string, startPos, endPos int) (string, bool) {
		if !strings.HasPrefix(s, fleet.ServerSecretPrefix) {
			return "", false
		}
		val, ok := secretMap[strings.TrimPrefix(s, fleet.ServerSecretPrefix)]

		switch {
		case documentIsJSON:
			val = jsonEscapeString(val)
		case documentIsXML:
			var b strings.Builder
			err = xml.EscapeText(&b, []byte(val))
			if err != nil {
				return "", false
			}
			val = b.String()
		}

		return val, ok
	})

	return expanded, secrets, nil
}

// jsonEscapeString returns the JSON-escaped interior of a string value
// (without surrounding quotes), suitable for embedding inside a JSON string.
func jsonEscapeString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		// json.Marshal on a string should never fail, but return the
		// original string as a fallback.
		return s
	}
	return string(b[1 : len(b)-1])
}

func (ds *Datastore) ExpandEmbeddedSecretsAndUpdatedAt(ctx context.Context, document string) (string, *time.Time, error) {
	expanded, secrets, err := ds.expandEmbeddedSecrets(ctx, document)
	if err != nil {
		return "", nil, ctxerr.Wrap(ctx, err, "expanding embedded secrets and updated at")
	}
	if len(secrets) == 0 {
		return expanded, nil, nil
	}
	// Find the most recent updated_at timestamp
	var updatedAt time.Time
	for _, secret := range secrets {
		if secret.UpdatedAt.After(updatedAt) {
			updatedAt = secret.UpdatedAt
		}
	}
	return expanded, &updatedAt, err
}

func (ds *Datastore) ValidateEmbeddedSecrets(ctx context.Context, documents []string) error {
	wantSecrets := make(map[string]struct{})
	haveSecrets := make(map[string]struct{})

	for _, document := range documents {
		vars := fleet.ContainsPrefixVars(document, fleet.ServerSecretPrefix)
		for _, v := range vars {
			wantSecrets[v] = struct{}{}
		}
	}

	wantSecretsList := make([]string, 0, len(wantSecrets))
	for wantSecret := range wantSecrets {
		wantSecretsList = append(wantSecretsList, wantSecret)
	}

	dbSecrets, err := ds.GetSecretVariables(ctx, wantSecretsList)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating document embedded secrets")
	}

	for _, dbSecret := range dbSecrets {
		haveSecrets[dbSecret.Name] = struct{}{}
	}

	missingSecrets := []string{}

	for wantSecret := range wantSecrets {
		if _, ok := haveSecrets[wantSecret]; !ok {
			missingSecrets = append(missingSecrets, wantSecret)
		}
	}

	if len(missingSecrets) > 0 {
		return &fleet.MissingSecretsError{MissingSecrets: missingSecrets}
	}

	return nil
}

// ExpandHostSecrets expands host-scoped secrets ($FLEET_HOST_SECRET_*) in the document.
// The enrollmentID (typically UDID/host UUID) is used to look up host-specific secrets.
func (ds *Datastore) ExpandHostSecrets(ctx context.Context, document string, enrollmentID string) (string, error) {
	// Check for host secret placeholders
	hostSecrets := fleet.ContainsPrefixVars(document, fleet.HostSecretPrefix)
	if len(hostSecrets) == 0 {
		return document, nil
	}

	// Build a map of secret type -> value
	// enrollmentID is the host UUID, which is the primary key in host_recovery_key_passwords
	secretValues := make(map[string]string)
	for _, secretType := range hostSecrets {
		switch secretType {
		case fleet.HostSecretRecoveryLockPassword:
			password, err := ds.getHostRecoveryLockPasswordDecrypted(ctx, enrollmentID)
			if err != nil {
				return "", ctxerr.Wrapf(ctx, err, "getting recovery lock password for host %s", enrollmentID)
			}
			secretValues[secretType] = password
		case fleet.HostSecretRecoveryLockPendingPassword:
			password, err := ds.getHostRecoveryLockPendingPasswordDecrypted(ctx, enrollmentID)
			if err != nil {
				return "", ctxerr.Wrapf(ctx, err, "getting pending recovery lock password for host %s", enrollmentID)
			}
			secretValues[secretType] = password
		case fleet.HostSecretMDMUnlockToken:
			details, err := ds.GetNanoMDMEnrollmentDetails(ctx, enrollmentID)
			if err != nil {
				return "", ctxerr.Wrapf(ctx, err, "getting MDM enrollment details for host %s", enrollmentID)
			}
			if details == nil {
				return "", ctxerr.Errorf(ctx, "no MDM enrollment details found for host %s", enrollmentID)
			}
			if details.UnlockToken == nil {
				return "", ctxerr.Errorf(ctx, "%s", fleet.CantClearPasscodePersonalHostsMessage)
			}

			// We need to send base64 encoded data in the <data> field.
			encoded := base64.StdEncoding.EncodeToString([]byte(*details.UnlockToken))
			secretValues[secretType] = encoded
		default:
			return "", ctxerr.Errorf(ctx, "unknown host secret type: %s", secretType)
		}
	}

	// Detect document format (same logic as expandEmbeddedSecrets)
	trimmed := strings.TrimSpace(document)
	documentIsXML := strings.HasPrefix(trimmed, "<")
	documentIsJSON := strings.HasPrefix(trimmed, "{")

	// Expand the placeholders
	expanded := fleet.MaybeExpand(document, func(s string, startPos, endPos int) (string, bool) {
		if !strings.HasPrefix(s, fleet.HostSecretPrefix) {
			return "", false
		}
		secretType := strings.TrimPrefix(s, fleet.HostSecretPrefix)
		val, ok := secretValues[secretType]
		if !ok {
			return "", false
		}

		switch {
		case documentIsJSON:
			val = jsonEscapeString(val)
		case documentIsXML:
			var b strings.Builder
			if err := xml.EscapeText(&b, []byte(val)); err != nil {
				return "", false
			}
			val = b.String()
		}

		return val, ok
	})

	return expanded, nil
}

// getHostRecoveryLockPasswordDecrypted retrieves and decrypts the recovery lock password for a host.
func (ds *Datastore) getHostRecoveryLockPasswordDecrypted(ctx context.Context, hostUUID string) (string, error) {
	var encryptedPassword []byte
	err := sqlx.GetContext(ctx, ds.reader(ctx), &encryptedPassword,
		`SELECT encrypted_password FROM host_recovery_key_passwords WHERE host_uuid = ? AND deleted = 0`, hostUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ctxerr.Wrap(ctx, notFound("HostRecoveryLockPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return "", ctxerr.Wrap(ctx, err, "getting encrypted recovery lock password")
	}

	password, err := decrypt(encryptedPassword, ds.serverPrivateKey)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decrypting recovery lock password")
	}

	return string(password), nil
}

// getHostRecoveryLockPendingPasswordDecrypted retrieves and decrypts the pending recovery lock
// password for a host during password rotation.
func (ds *Datastore) getHostRecoveryLockPendingPasswordDecrypted(ctx context.Context, hostUUID string) (string, error) {
	var encryptedPassword []byte
	err := sqlx.GetContext(ctx, ds.reader(ctx), &encryptedPassword,
		`SELECT pending_encrypted_password FROM host_recovery_key_passwords WHERE host_uuid = ? AND deleted = 0 AND pending_encrypted_password IS NOT NULL`, hostUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ctxerr.Wrap(ctx, notFound("HostRecoveryLockPendingPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return "", ctxerr.Wrap(ctx, err, "getting encrypted pending recovery lock password")
	}

	password, err := decrypt(encryptedPassword, ds.serverPrivateKey)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decrypting pending recovery lock password")
	}

	return string(password), nil
}
