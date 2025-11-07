package mysql

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/jmoiron/sqlx"
)

// Most of the code for the host identity feature is located at ./ee/server/service/hostidentity

func (ds *Datastore) GetHostIdentityCertBySerialNumber(ctx context.Context, serialNumber uint64) (*types.HostIdentityCertificate, error) {
	var hostIdentityCert types.HostIdentityCertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostIdentityCert, fmt.Sprintf(`
		SELECT serial, host_id, name, not_valid_after, public_key_raw
		FROM host_identity_scep_certificates
		WHERE serial = %d
			AND not_valid_after > NOW()
			AND revoked = 0`, serialNumber))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("host identity certificate")
	case err != nil:
		return nil, err
	}
	return &hostIdentityCert, nil
}

func (ds *Datastore) UpdateHostIdentityCertHostIDBySerial(ctx context.Context, serialNumber uint64, hostID uint) error {
	return common_mysql.WithRetryTxx(ctx, ds.writer(ctx), func(tx sqlx.ExtContext) error {
		return updateHostIdentityCertHostIDBySerial(ctx, tx, hostID, serialNumber)
	}, ds.logger)
}

func updateHostIdentityCertHostIDBySerial(ctx context.Context, tx sqlx.ExtContext, hostID uint, serialNumber uint64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE host_identity_scep_certificates
		SET host_id = ?
		WHERE serial = ?`, hostID, serialNumber)
	return err
}

func (ds *Datastore) GetHostIdentityCertByName(ctx context.Context, name string) (*types.HostIdentityCertificate, error) {
	var hostIdentityCert types.HostIdentityCertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostIdentityCert, `
		SELECT serial, host_id, name, not_valid_after, public_key_raw, created_at
		FROM host_identity_scep_certificates
		WHERE name = ?
			AND not_valid_after > NOW()
			AND revoked = 0`, name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("host identity certificate")
	case err != nil:
		return nil, err
	}
	return &hostIdentityCert, nil
}

// GetMDMSCEPCertBySerial looks up an MDM SCEP certificate by serial number
// and returns the device UUID it's associated with. This is used for iOS/iPadOS
// certificate-based authentication on the My Device page.
//
// This query uses the nano_cert_auth_associations table which maps device IDs to
// certificate hashes. The serial number lookup in scep_certificates provides
// the raw certificate data, but we need the nanomdm association to get the device UUID.
func (ds *Datastore) GetMDMSCEPCertBySerial(ctx context.Context, serialNumber uint64) (deviceUUID string, err error) {
	// First get the certificate by serial
	var certPEM string
	err = sqlx.GetContext(ctx, ds.reader(ctx), &certPEM, `
		SELECT certificate_pem
		FROM scep_certificates
		WHERE serial = ?
			AND not_valid_after > NOW()
			AND revoked = 0`, serialNumber)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", notFound("MDM SCEP certificate")
	case err != nil:
		return "", err
	}

	// Calculate the SHA256 hash of the certificate the same way nanomdm does
	// (see server/mdm/nanomdm/service/certauth/certauth.go HashCert function)
	// The hash is calculated from cert.Raw (DER-encoded bytes), not the PEM string
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}
	hashed := sha256.Sum256(cert.Raw)
	hash := hex.EncodeToString(hashed[:])

	// Look up the device UUID by certificate hash
	err = sqlx.GetContext(ctx, ds.reader(ctx), &deviceUUID, `
		SELECT id
		FROM nano_cert_auth_associations
		WHERE sha256 = ?`, hash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", notFound("MDM certificate association")
	case err != nil:
		return "", err
	}
	return deviceUUID, nil
}
