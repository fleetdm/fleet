package mysql

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"math/big"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
)

// SCEPDepot is a MySQL-backed SCEP certificate depot.
type SCEPDepot struct {
	db *sql.DB
	ds fleet.Datastore
}

var _ depot.Depot = (*SCEPDepot)(nil)

// newSCEPDepot creates and returns a *SCEPDepot.
func newSCEPDepot(db *sql.DB, ds fleet.Datastore) (*SCEPDepot, error) {
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &SCEPDepot{
		db: db,
		ds: ds,
	}, nil
}

// CA returns the CA's certificate and private key.
func (d *SCEPDepot) CA(_ []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	// TODO(roberto): nano interfaces doesn't receive a context for this method.
	cert, err := assets.CAKeyPair(context.Background(), d.ds)
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
func (d *SCEPDepot) Serial() (*big.Int, error) {
	result, err := d.db.Exec(`INSERT INTO scep_serials () VALUES ();`)
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
//
// TODO(lucas): Implement and use allowTime and revokeOldCertificate.
// - allowTime are the maximum days before expiration to allow clients to do certificate renewal.
// - revokeOldCertificate specifies whether to revoke the old certificate once renewed.
func (d *SCEPDepot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
	var ct int
	row := d.db.QueryRow(`SELECT COUNT(*) FROM scep_certificates WHERE name = ?`, cn)
	if err := row.Scan(&ct); err != nil {
		return false, err
	}
	return ct >= 1, nil
}

// Put stores a certificate under the given name.
//
// If the provided certificate has empty crt.Subject.CommonName,
// then the hex sha256 of the crt.Raw is used as name.
func (d *SCEPDepot) Put(name string, crt *x509.Certificate) error {
	if crt.Subject.CommonName == "" {
		name = fmt.Sprintf("%x", sha256.Sum256(crt.Raw))
	}
	if !crt.SerialNumber.IsInt64() {
		return errors.New("cannot represent serial number as int64")
	}
	certPEM := apple_mdm.EncodeCertPEM(crt)
	_, err := d.db.Exec(`
INSERT INTO scep_certificates
    (serial, name, not_valid_before, not_valid_after, certificate_pem)
VALUES
    (?, ?, ?, ?, ?)`,
		crt.SerialNumber.Int64(),
		name,
		crt.NotBefore,
		crt.NotAfter,
		certPEM,
	)
	return err
}
