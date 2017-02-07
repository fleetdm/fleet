package datastore

import (
	"math/rand"
	"testing"
	"time"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var token = "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjRkOmM1OmRlOmE1OjczOm" +
	"UxOmE4OjI4OmU2OmEyOjMwOmI4OmI1OjBmOjg4OjQ0In0.eyJsaWNlbnNlX3V1aWQiOiIyYWYyZD" +
	"lhMC1iOWE1LTQ0ZTItODU1NC04Mjc2MGI4ODQwZDYiLCJvcmdhbml6YXRpb25fbmFtZSI6IlBoYW5" +
	"0YXNtLCBJbmMuIiwib3JnYW5pemF0aW9uX3V1aWQiOiJkZmJkNWIwMy0xMDg0LTQ2YWUtYjM4MS1l" +
	"MTI5YWM2NmU4ZDgiLCJob3N0X2xpbWl0IjowLCJldmFsdWF0aW9uIjp0cnVlLCJleHBpcmVzX2F0I" +
	"joiMjAxNy0wMy0wNFQxNTowMTo0OSswMDowMCJ9.Ny4Fxqlq_4U647gmIouFPZQH4YG8R_AHOlDTB" +
	"ObWOUfhcKiz44vRkCqr_Jqprb0zVtSVy1bMojLLmQhKjSxQZuiqvQBfou9Osfd5D3i-TXEb5JpoCg" +
	"Fem-1t5jvOT7T9H4HJpuKE40cnOl3Zu2OzjjdxMMZbj_i2iwZytW1b7SrGNAwJVXXwJs2a95bGbMu" +
	"ZWyV-YpuHaWlx-VpTv4c2vQo2eQWTpTH7YdcQ7Mo_5QdN7247qKo_ORTtqLLTjg7BoxB__ydWMhxO" +
	"QuRJGQAMc0OsZ72uLd7JKzvWpSLFk7mdVk718mweq6X2R0BPKtTc6lYjbPScoTysM2Owe5Hi7A"

func testLicense(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated")
	}
	err := ds.MigrateData()
	require.Nil(t, err)
	license, err := ds.License()
	require.Nil(t, err)
	assert.Nil(t, license.Token)

	publicKey, err := ds.LicensePublicKey(token)
	require.Nil(t, err)

	_, err = ds.SaveLicense(token, publicKey)
	require.Nil(t, err)
	license, err = ds.License()
	require.Nil(t, err)
	require.NotNil(t, license.Token)
	assert.Equal(t, token, *license.Token)

	err = ds.RevokeLicense(!license.Revoked)
	require.Nil(t, err)
	changedLicense, err := ds.License()
	require.Nil(t, err)
	assert.NotEqual(t, license.Revoked, changedLicense.Revoked)

	// screw around with the token in random ways and make sure that A) it doesn't
	// panic and B) returns an error
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 10; i++ {
		j := r.Uint32()
		k := j % uint32(len(token))
		buff := []byte(token)
		buff[k] = buff[k] + 1
		_, err = ds.LicensePublicKey(string(buff))
		require.NotNil(t, err)
	}

}
