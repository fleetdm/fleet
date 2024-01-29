package pgsql

import (
	"context"
	"crypto/tls"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
)

func (s *PgSQLStorage) RetrievePushCert(ctx context.Context, topic string) (*tls.Certificate, string, error) {
	var certPEM, keyPEM []byte
	var staleToken int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT cert_pem, key_pem, stale_token FROM push_certs WHERE topic = $1;`,
		topic,
	).Scan(&certPEM, &keyPEM, &staleToken)
	if err != nil {
		return nil, "", err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, "", err
	}
	return &cert, strconv.Itoa(staleToken), err
}

func (s *PgSQLStorage) IsPushCertStale(ctx context.Context, topic, staleToken string) (bool, error) {
	var staleTokenInt, dbStaleToken int
	staleTokenInt, err := strconv.Atoi(staleToken)
	if err != nil {
		return true, err
	}
	err = s.db.QueryRowContext(
		ctx,
		`SELECT stale_token FROM push_certs WHERE topic = $1;`,
		topic,
	).Scan(&dbStaleToken)
	return dbStaleToken != staleTokenInt, err
}

func (s *PgSQLStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx, `
INSERT INTO push_certs
    (topic, cert_pem, key_pem, stale_token)
VALUES
    ($1, $2, $3, 0)
ON CONFLICT (topic) DO
UPDATE SET
    cert_pem = EXCLUDED.cert_pem,
    key_pem = EXCLUDED.key_pem,
    stale_token = push_certs.stale_token + 1;`,
		topic, pemCert, pemKey,
	)
	return err
}
