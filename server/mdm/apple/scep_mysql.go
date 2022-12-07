package apple_mdm

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/scep/v2/depot"
)

// SCEPMySQLDepot is a MySQL-backed SCEP certificate depot.
type SCEPMySQLDepot struct {
	db *sql.DB

	// caCrt holds the CA's certificate.
	caCrt *x509.Certificate
	// caKey holds the CA private key.
	caKey *rsa.PrivateKey
}

var _ depot.Depot = (*SCEPMySQLDepot)(nil)

// NewSCEPMySQLDepot creates and returns a *SCEPMySQLDepot.
func NewSCEPMySQLDepot(db *sql.DB, caCertPEM []byte, caKeyPEM []byte) (*SCEPMySQLDepot, error) {
	if err := db.Ping(); err != nil {
		return nil, err
	}
	caCrt, err := cryptoutil.DecodePEMCertificate(caCertPEM)
	if err != nil {
		return nil, err
	}
	caKey, err := decodeRSAKeyFromPEM(caKeyPEM)
	if err != nil {
		return nil, err
	}
	return &SCEPMySQLDepot{
		db:    db,
		caCrt: caCrt,
		caKey: caKey,
	}, nil
}

func decodeRSAKeyFromPEM(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("PEM type is not RSA PRIVATE KEY")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// CA returns the CA's certificate and private key.
func (d *SCEPMySQLDepot) CA(_ []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	return []*x509.Certificate{d.caCrt}, d.caKey, nil
}

// Serial allocates and returns a new (increasing) serial number.
func (d *SCEPMySQLDepot) Serial() (*big.Int, error) {
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
func (d *SCEPMySQLDepot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
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
func (d *SCEPMySQLDepot) Put(name string, crt *x509.Certificate) error {
	if crt.Subject.CommonName == "" {
		name = fmt.Sprintf("%x", sha256.Sum256(crt.Raw))
	}
	if !crt.SerialNumber.IsInt64() {
		return errors.New("cannot represent serial number as int64")
	}
	certPEM := EncodeCertPEM(crt)
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
