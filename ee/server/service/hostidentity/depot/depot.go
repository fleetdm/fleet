package depot

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

const maxCommonNameLength = 255

// HostIdentitySCEPDepot is a MySQL-backed SCEP certificate depot.
type HostIdentitySCEPDepot struct {
	db     *sqlx.DB
	ds     fleet.Datastore
	logger log.Logger
	config *config.FleetConfig
}

var _ depot.Depot = (*HostIdentitySCEPDepot)(nil)

// NewHostIdentitySCEPDepot creates and returns a *HostIdentitySCEPDepot.
func NewHostIdentitySCEPDepot(db *sqlx.DB, ds fleet.Datastore, logger log.Logger, cfg *config.FleetConfig) (*HostIdentitySCEPDepot, error) {
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &HostIdentitySCEPDepot{
		db:     db,
		ds:     ds,
		logger: logger,
		config: cfg,
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
	// Insert an empty row to generate a new auto-incremented serial number
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
func (d *HostIdentitySCEPDepot) Put(name string, crt *x509.Certificate) error {
	if crt.Subject.CommonName == "" || len(crt.Subject.CommonName) > maxCommonNameLength {
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
	pubKeyRaw, err := types.CreateECDSAPublicKeyRaw(key)
	if err != nil {
		return fmt.Errorf("creating public key raw: %w", err)
	}
	certPEM := certificate.EncodeCertPEM(crt)

	// Apply rate limiting if configured
	cooldown := d.config.Osquery.EnrollCooldown
	if cooldown > 0 {
		existingCert, err := d.ds.GetHostIdentityCertByName(context.Background(), name)
		switch {
		case err != nil && !fleet.IsNotFound(err):
			return fmt.Errorf("checking existing certificate: %w", err)
		case err == nil:
			// Certificate exists, check if rate limit applies
			if time.Since(existingCert.CreatedAt) < cooldown {
				return backoff.Permanent(ctxerr.Errorf(context.Background(), "host identified by %s requesting certificates too often", name))
			}
		}
		// If certificate doesn't exist or rate limit doesn't apply, continue
	}

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
