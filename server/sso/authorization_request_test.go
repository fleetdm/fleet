package sso

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestCompression(t *testing.T) {
	input := "<samlp:AuthnRequest AssertionConsumerServiceURL='https://sp.example.com/acs' Destination='https://idp.example.com/sso' ID='_18185425-fd62-477c-b9d4-4b5d53a89845' IssueInstant='2017-04-16T15:32:42Z' ProtocolBinding='urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST' Version='2.0' xmlns:saml='urn:oasis:names:tc:SAML:2.0:assertion' xmlns:samlp='urn:oasis:names:tc:SAML:2.0:protocol'><saml:Issuer>https://sp.example.com/saml2</saml:Issuer><samlp:NameIDPolicy AllowCreate='true' Format='urn:oasis:names:tc:SAML:2.0:nameid-format:transient'/></samlp:AuthnRequest>"
	expected := "fJJf79IwFIa/Su961f0pG4yGLZkQ4xLUBaYX3piyHaTJ2s6eTvHbmw2McPHjtnne9u1zzgal7gdRjv5iDvBzBPSkRATnlTVba3DU4I7gfqkWvhz2Ob14P6AIQxwCuEo99BC0VoeyRUp2gF4ZOUX/g6p7JhEtJdUup9/jLM7ShKfs3C05S1arlp3WXcKSU9qlC5mtsySlpEIcoTLopfE55VG8YlHC4mUTp2LBRcK/UVI7621r+3fKdMr8yOnojLASFQojNaDwrTiWH/eCB5E43SAUH5qmZvXnY0PJV3A4t+ZBRMlV9wbFZOb1TfKfqMfI8Doz3KvSYlYv5u+54g2tE8I34SN5n9gnqaHa1bZX7R9S9r39vXUgPeTUuxEoeW+dlv51l+lEdew8o8I7aVCB8TQsbk8+70XxNwAA//8="
	buff := bytes.NewBufferString(input)
	compressed, err := deflate(buff)
	require.Nil(t, err)
	assert.Equal(t, expected, compressed)
}

func TestCreateAuthorizationRequest(t *testing.T) {
}
