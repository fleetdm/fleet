package mysql

import (
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (ds *Datastore) RevokeLicense(revoked bool) error {
	sql := `
		UPDATE licenses SET
			revoked = ?
		WHERE id = 1
	`
	results, err := ds.db.Exec(sql, revoked)
	if err != nil {
		return errors.Wrap(err, "updating license revoked")
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected updating license revoked")
	}
	if rowsAffected == 0 {
		return notFound("License").WithID(uint(1))
	}
	return nil
}

// LicensePublicKey will insure that a jwt token is signed properly and that we
// have the public key we need to validate it.  The public key string is returned
// on success
func (ds *Datastore) LicensePublicKey(token string) (string, error) {
	var (
		publicKeyHash string
		publicKey     string
	)
	_, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return "", fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		h, ok := token.Header["kid"]
		if !ok {
			return "", errors.New("missing kid header")
		}
		publicKeyHash, ok = h.(string)
		if !ok {
			return "", errors.New("kid is not expected type")
		}

		sql := `
			SELECT pk.key
				FROM public_keys pk
				WHERE hash = ?`

		err := ds.db.Get(&publicKey, sql, publicKeyHash)
		if err != nil {
			return "", errors.Wrap(err, "could not find public key matching hash")
		}
		return jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
	})
	return publicKey, err
}

func (ds *Datastore) SaveLicense(token, publicKey string) (*kolide.License, error) {
	sqlStatement := `
		 UPDATE licenses SET
			token = ?,
			` + "`key`" + ` = ?
		 WHERE id = 1`

	res, err := ds.db.Exec(sqlStatement, token, publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "saving license")
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, errors.Wrap(err, "rows affected saving license")
	}
	if rowsAffected == 0 {
		return nil, notFound("License").WithID(uint(1))
	}
	result, err := ds.License()
	if err != nil {
		return nil, errors.Wrap(err, "fetching license")
	}
	return result, nil
}

func (ds *Datastore) License() (*kolide.License, error) {
	query := `
	  SELECT * FROM licenses
	    WHERE id = 1
	  `
	var license kolide.License
	err := ds.db.Get(&license, query)
	if err != nil {
		return nil, errors.Wrap(err, "fetching license information")
	}
	query = `
    SELECT count(*)
      FROM hosts
      WHERE NOT deleted
  `
	err = ds.db.Get(&license.HostCount, query)
	if err != nil {
		return nil, errors.Wrap(err, "fetching host count for license")
	}
	return &license, nil
}
