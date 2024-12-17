package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) UpsertSecretVariables(ctx context.Context, secretVariables []fleet.SecretVariable) error {
	if len(secretVariables) == 0 {
		return nil
	}

	values := strings.TrimSuffix(strings.Repeat("(?,?),", len(secretVariables)), ",")

	stmt := fmt.Sprintf(`
		INSERT INTO secret_variables (name, value)
		VALUES %s
		ON DUPLICATE KEY UPDATE value = VALUES(value)`, values)

	args := make([]interface{}, 0, len(secretVariables)*2)
	for _, secretVariable := range secretVariables {
		valueEncrypted, err := encrypt([]byte(secretVariable.Value), ds.serverPrivateKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypt secret value with server private key")
		}
		args = append(args, secretVariable.Name, valueEncrypted)
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert secret variables")
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

func (ds *Datastore) ExpandEmbeddedSecrets(ctx context.Context, document string) (string, error) {
	embeddedSecrets := fleet.ContainsPrefixVars(document, fleet.ServerSecretPrefix)

	secrets, err := ds.GetSecretVariables(ctx, embeddedSecrets)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "expanding embedded secrets")
	}

	secretMap := make(map[string]string, len(secrets))

	for _, secret := range secrets {
		secretMap[secret.Name] = secret.Value
	}

	missingSecrets := []string{}

	for _, wantSecret := range embeddedSecrets {
		if _, ok := secretMap[wantSecret]; !ok {
			missingSecrets = append(missingSecrets, fmt.Sprintf("$FLEET_SECRET_%s", wantSecret))
		}
	}

	if len(missingSecrets) > 0 {
		return "", ctxerr.Errorf(ctx, "embedded secrets missing from datastore: %s", strings.Join(missingSecrets, ", "))
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
