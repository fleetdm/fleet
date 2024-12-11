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
