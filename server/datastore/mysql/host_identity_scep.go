package mysql

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

// HostIdentitySCEPDepot is a MySQL-backed SCEP certificate depot.
type HostIdentitySCEPDepot struct {
	db     *sqlx.DB
	ds     fleet.Datastore
	logger log.Logger
}

var _ depot.Depot = (*HostIdentitySCEPDepot)(nil)

// newHostIdentitySCEPDepot creates and returns a *HostIdentitySCEPDepot.
func newHostIdentitySCEPDepot(db *sqlx.DB, ds fleet.Datastore, logger log.Logger) (*HostIdentitySCEPDepot, error) {
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &HostIdentitySCEPDepot{
		db:     db,
		ds:     ds,
		logger: logger,
	}, nil
}

// CA returns the CA's certificate and private key.
func (d *HostIdentitySCEPDepot) CA(_ []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := assets.KeyPair(context.Background(), d.ds, fleet.MDMAssetHostIdentityCACert, fleet.MDMAssetHostIdentityCAKey)
	if err != nil {
		return nil, nil, fmt.Errorf("getting assets: %w", err)
	}

	pk, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("private key not in RSA format")
	}

	return []*x509.Certificate{cert.Leaf}, pk, nil
}

// Serial allocates and returns a new (increasing) serial number.
func (d *HostIdentitySCEPDepot) Serial() (*big.Int, error) {
	result, err := d.db.Exec(`INSERT INTO host_identity_scep_serials () VALUES ();`)
	if err != nil {
		return nil, err
	}
	lid, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return big.NewInt(lid), nil
}

// HasCN returns whether the given certificate exists in the depot.
func (d *HostIdentitySCEPDepot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
	// Not used right now. May be used for renewal in the future.
	return false, nil
}

// Put stores a certificate under the given name.
//
// If the provided certificate has empty crt.Subject.CommonName,
// then the hex sha256 of the crt.Raw is used as name.
func (d *HostIdentitySCEPDepot) Put(name string, crt *x509.Certificate) error {
	const maxCNLength = 255
	if crt.Subject.CommonName == "" || len(crt.Subject.CommonName) > maxCNLength {
		return errors.New("common name empty or too long")
	}
	if !crt.SerialNumber.IsInt64() {
		return errors.New("cannot represent serial number as int64")
	}

	// Extract the ECC uncompressed point (04-prefixed X || Y); 0x04 means this is the raw representation
	// Lengths:
	//   - P-256: 65 bytes
	//   - P-384: 97 bytes
	key, ok := crt.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("public key not in ECDSA format")
	}
	pubKeyRaw, err := fleet.CreateECDSAPublicKeyRaw(key)
	if err != nil {
		return fmt.Errorf("creating public key raw: %w", err)
	}
	certPEM := certificate.EncodeCertPEM(crt)

	return common_mysql.WithRetryTxx(context.Background(), d.db, func(tx sqlx.ExtContext) error {
		// Revoke existing certs for this host id.
		// Note: Because the challenge is shared, it is possible for a bad actor to revoke a cert for an existing host
		// if they have the challenge and the host identifier (CN).
		result, err := tx.ExecContext(context.Background(), `
			UPDATE host_identity_scep_certificates 
			SET revoked = 1 
			WHERE name = ?`, name)
		if err != nil {
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			d.logger.Log("msg", "revoked existing host identity certificate", "name", name)
		}

		_, err = tx.ExecContext(context.Background(), `
			INSERT INTO host_identity_scep_certificates
				(serial, name, not_valid_before, not_valid_after, certificate_pem, public_key_raw)
			VALUES
				(?, ?, ?, ?, ?, ?)`,
			crt.SerialNumber.Int64(),
			name,
			crt.NotBefore,
			crt.NotAfter,
			certPEM,
			pubKeyRaw,
		)
		return err
	}, d.logger)
}

func (ds *Datastore) GetHostIdentityCertBySerialNumber(ctx context.Context, serialNumber uint64) (*fleet.HostIdentityCertificate, error) {
	var hostIdentityCert fleet.HostIdentityCertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostIdentityCert, `
		SELECT serial, host_id, name, not_valid_after, public_key_raw
		FROM host_identity_scep_certificates
		WHERE serial = ?
			AND not_valid_after > NOW()
			AND revoked = 0`, serialNumber)
	if err != nil {
		return nil, err
	}
	return &hostIdentityCert, nil
}
