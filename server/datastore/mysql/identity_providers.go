package mysql

import (
	"database/sql"

	"github.com/kolide/kolide/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewIdentityProvider(idp kolide.IdentityProvider) (*kolide.IdentityProvider, error) {
	query := `
    INSERT INTO identity_providers (
      sso_url,
      issuer_uri,
      cert,
      name,
      image_url
    )
    VALUES ( ?, ?, ?, ?, ? )
  `
	result, err := d.db.Exec(query, idp.SingleSignOnURL, idp.IssuerURI, idp.Certificate, idp.Name, idp.ImageURL)
	if err != nil {
		return nil, errors.Wrap(err, "creating identity provider")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving id for new identity provider")
	}
	idp.ID = uint(id)
	return &idp, nil
}

func (d *Datastore) SaveIdentityProvider(idp kolide.IdentityProvider) error {
	query := `
    UPDATE identity_providers
    SET
      sso_url = ?,
      issuer_uri = ?,
      cert = ?,
      name = ?,
      image_url = ?
    WHERE id = ?
  `
	result, err := d.db.Exec(query, idp.SingleSignOnURL, idp.IssuerURI, idp.Certificate,
		idp.Name, idp.ImageURL, idp.ID)
	if err != nil {
		return errors.Wrap(err, "updating identity provider")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "fetching updated row count for identity provider")
	}
	if rows == 0 {
		return notFound("IdentityProvider").WithID(idp.ID)
	}
	return nil
}

func (d *Datastore) IdentityProvider(id uint) (*kolide.IdentityProvider, error) {
	query := `
    SELECT *
    FROM identity_providers
    WHERE id = ? AND NOT deleted
  `
	var idp kolide.IdentityProvider
	err := d.db.Get(&idp, query, id)
	if err == sql.ErrNoRows {
		return nil, notFound("IdentityProvider").WithID(id)
	}
	if err != nil {
		return nil, errors.Wrap(err, "selecting identity provider")
	}
	return &idp, nil
}

func (d *Datastore) DeleteIdentityProvider(id uint) error {
	return d.deleteEntity("identity_providers", id)
}

func (d *Datastore) ListIdentityProviders() ([]kolide.IdentityProvider, error) {
	query := `
    SELECT *
    FROM identity_providers
    WHERE NOT deleted
  `
	var idps []kolide.IdentityProvider
	if err := d.db.Select(&idps, query); err != nil {
		return nil, errors.Wrap(err, "listing identity providers")
	}
	return idps, nil
}
