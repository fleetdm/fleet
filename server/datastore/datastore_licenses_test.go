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

var publicKey = `-----BEGIN PUBLIC KEY-----
 MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0ZhY7r6HmifXPtServt4
 D3MSi8Awe9u132vLf8yzlknvnq+8CSnOPSSbCD+HajvZ6dnNJXjdcAhuZ32ShrH8
 rEQACEUS8Mh4z8Mo5Nlq1ou0s2JzWCx049kA34jP3u6AiPgpWUf8JRGstTlisxMn
 H6B7miDs1038gVbN5rk+j+3ALYzllaTnCX3Y0C7f6IW7BjNO/tvFB84/95xfOLEz
 o2MeFMqkD29hvcrUW+8+fQGJaVLvcEqBDnIEVbCCk8Wnoi48dUE06WHUl6voJecD
 dW1E6jHcq8PQFK+4bI1gKZVbV4dFGSSMUyD7ov77aWHjxdQe6YEGcSXKzfyMaUtQ
 vQIDAQAB
 -----END PUBLIC KEY-----
 `

func testLicense(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is deprecated")
	}

	err := ds.MigrateData()
	require.Nil(t, err)
	license, err := ds.License()
	require.Nil(t, err)
	assert.Nil(t, license.Token)

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
