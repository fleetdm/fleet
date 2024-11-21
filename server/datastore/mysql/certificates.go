package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetPKICertificate(ctx context.Context, name string) (*fleet.PKICertificate, error) {
	stmt := `
		SELECT name, cert_pem, key_pem, sha256_hex, not_valid_after
		FROM pki_certificates
		WHERE name = ?
	`
	var cert fleet.PKICertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &cert, stmt, name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("pki certificate").WithName(name)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "get pki certificate")
	}
	if len(cert.Cert) > 0 {
		cert.Cert, err = decrypt(cert.Cert, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decrypting pki cert")
		}
	}
	if len(cert.Key) > 0 {
		cert.Key, err = decrypt(cert.Key, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decrypting pki key")
		}
	}
	return &cert, nil
}

func (ds *Datastore) SavePKICertificate(ctx context.Context, cert *fleet.PKICertificate) error {
	const stmt = `
		INSERT INTO
			pki_certificates (name, cert_pem, key_pem, not_valid_after, sha256)
		VALUES
			(?, ?, ?, ?, UNHEX(?))
		ON DUPLICATE KEY UPDATE
			cert_pem = VALUES(cert_pem),
			key_pem = VALUES(key_pem),
			not_valid_after = VALUES(not_valid_after),
			sha256 = VALUES(sha256)
`
	var err error
	var encryptedCert, encryptedKey []byte
	if len(cert.Cert) > 0 {
		encryptedCert, err = encrypt(cert.Cert, ds.serverPrivateKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypting pki cert")
		}
	}
	if len(cert.Key) > 0 {
		encryptedKey, err = encrypt(cert.Key, ds.serverPrivateKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypting pki key")
		}
	}
	_, err = ds.writer(ctx).ExecContext(ctx, stmt, cert.Name, encryptedCert, encryptedKey, cert.NotValidAfter, cert.Sha256)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save pki certificate")
	}
	return nil
}

func (ds *Datastore) ListPKICertificates(ctx context.Context) ([]fleet.PKICertificate, error) {
	// Since name is the primary key, results are always sorted by name
	stmt := `
		SELECT name, not_valid_after, sha256_hex
		FROM pki_certificates
	`
	var certs []fleet.PKICertificate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &certs, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pki certificates")
	}
	return certs, nil
}

func (ds *Datastore) DeletePKICertificate(ctx context.Context, name string) error {
	stmt := `
		DELETE FROM pki_certificates
		WHERE name = ?
	`
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, name)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete pki certificate")
	}
	return nil
}
