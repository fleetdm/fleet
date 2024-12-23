package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

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

func (ds *Datastore) GetSecretVariables(ctx context.Context, names []string) ([]fleet.SecretVariable, error) {
	if len(names) == 0 {
		return nil, nil
	}

	stmt, args, err := sqlx.In(`
		SELECT name, value
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

func (ds *Datastore) secretVariablesUpdated(ctx context.Context, q sqlx.QueryerContext, names []string, timestamp time.Time) (bool, error) {
	if len(names) == 0 {
		return false, nil
	}

	stmt, args, err := sqlx.In(`
		SELECT 1
		FROM secret_variables
		WHERE name IN (?) AND updated_at > ?`, names, timestamp)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "build secret variables query")
	}

	var updated bool
	err = sqlx.GetContext(ctx, q, &updated, stmt, args...)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, ctxerr.Wrap(ctx, err, "get secret variables")
	default:
		return updated, nil
	}
}

func (ds *Datastore) ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error) {
	embeddedSecrets := fleet.ContainsPrefixVars(document, fleet.ServerSecretPrefix)
	if len(embeddedSecrets) == 0 {
		return document, nil
	}

	secrets, err := ds.GetSecretVariables(ctx, embeddedSecrets)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "expanding embedded secrets")
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
		return "", fleet.MissingSecretsError{MissingSecrets: missingSecrets}
	}

	expanded := fleet.MaybeExpand(document, func(s string) (string, bool) {
		if !strings.HasPrefix(s, fleet.ServerSecretPrefix) {
			return "", false
		}
		val, ok := secretMap[strings.TrimPrefix(s, fleet.ServerSecretPrefix)]
		return val, ok
	})

	return expanded, nil
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
