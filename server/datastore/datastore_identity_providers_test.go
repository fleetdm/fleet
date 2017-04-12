package datastore

import (
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testIdentityProvider(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("imem is being deprecated")
	}
	idps := []*kolide.IdentityProvider{
		&kolide.IdentityProvider{
			SingleSignOnURL: "https://idp1.com/sso",
			IssuerURI:       "http://idp1.com/issuer/xyz123",
			Certificate:     "DEADBEEFXXXXX12344",
			Name:            "idp1",
			ImageURL:        "https://idp1.com/logo.png",
		},
		&kolide.IdentityProvider{
			SingleSignOnURL: "https://idp2.com/sso",
			IssuerURI:       "http://idp2.com/issuer/xyz123",
			Certificate:     "DEADBEEFXXXXX12344",
			Name:            "idp2",
			ImageURL:        "https://idp2.com/logo.png",
		},
		&kolide.IdentityProvider{
			SingleSignOnURL: "https://idp3.com/sso",
			IssuerURI:       "http://idp3.com/issuer/xyz123",
			Certificate:     "DEADBEEFXXXXX12344",
			Name:            "idp3",
			ImageURL:        "https://idp3.com/logo.png",
		},
	}
	var err error
	for i, idp := range idps {
		idps[i], err = ds.NewIdentityProvider(*idp)
		require.Nil(t, err)
		require.NotEqual(t, 0, idp.ID, "id assignment")
	}
	// duplicate name not allowed
	_, err = ds.NewIdentityProvider(*idps[0])
	assert.NotNil(t, err)
	// test get
	idp, err := ds.IdentityProvider(idps[0].ID)
	require.Nil(t, err)
	require.NotNil(t, idp)
	require.Equal(t, "idp1", idp.Name)
	// test update
	idp.ImageURL = "https://idpnew.com/logo.png"
	idp.SingleSignOnURL = "https://idpnew.com/sso"
	idp.IssuerURI = "https://idpnew.com/issuer"
	idp.Certificate = "123456789"
	idp.Name = "idpnew"
	err = ds.SaveIdentityProvider(*idp)
	require.Nil(t, err)
	upd, err := ds.IdentityProvider(idp.ID)
	require.Nil(t, err)
	require.NotNil(t, upd)
	assert.Equal(t, idp.ImageURL, upd.ImageURL)
	assert.Equal(t, idp.SingleSignOnURL, upd.SingleSignOnURL)
	assert.Equal(t, idp.IssuerURI, upd.IssuerURI)
	assert.Equal(t, idp.Certificate, upd.Certificate)
	assert.Equal(t, idp.Name, upd.Name)
	// test list
	results, err := ds.ListIdentityProviders()
	require.Nil(t, err)
	require.NotNil(t, results)
	assert.Len(t, results, 3)
	// test delete
	err = ds.DeleteIdentityProvider(results[0].ID)
	assert.Nil(t, err)
	err = ds.DeleteIdentityProvider(results[0].ID)
	assert.NotNil(t, err)
	results, err = ds.ListIdentityProviders()
	require.Nil(t, err)
	assert.NotNil(t, results, 2)
}
