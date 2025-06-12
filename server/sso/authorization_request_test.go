package sso

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
	store := &mockStore{}
	settings := &Settings{
		Metadata: &Metadata{
			EntityID: "test",
			IDPSSODescriptor: IDPSSODescriptor{
				SingleSignOnService: []SingleSignOnService{
					{Binding: RedirectBinding, Location: "http://example.com"},
				},
			},
		},
		// Construct call back url to send to idp
		AssertionConsumerServiceURL: "http://localhost:8001/api/v1/fleet/sso/callback",
		SessionStore:                store,
		OriginalURL:                 "/redir",
	}

	idpURL, err := CreateAuthorizationRequest(settings, "issuer", RelayState("abc"))
	require.NoError(t, err)

	parsed, err := url.Parse(idpURL)
	require.NoError(t, err)
	assert.Equal(t, "example.com", parsed.Host)
	q := parsed.Query()
	encoded := q.Get("SAMLRequest")
	assert.NotEmpty(t, encoded)
	authReq := inflate(t, encoded)
	assert.Equal(t, "issuer", authReq.Issuer.Url)
	assert.Equal(t, "Fleet", authReq.ProviderName)
	assert.True(t, strings.HasPrefix(authReq.ID, "id"), authReq.ID)
	assert.Equal(t, "abc", q.Get("RelayState"))

	ssn := store.session
	require.NotNil(t, ssn)
	assert.Equal(t, "/redir", ssn.OriginalURL)
	assert.Equal(t, 5*time.Minute, store.sessionLifetime)

	var meta Metadata
	require.NoError(t, xml.Unmarshal([]byte(ssn.Metadata), &meta))
	assert.Equal(t, "test", meta.EntityID)

	settings.SessionTTL = 3600
	_, err = CreateAuthorizationRequest(settings, "issuer", RelayState("abc"))
	require.NoError(t, err)
	assert.Equal(t, time.Hour, store.sessionLifetime)
}

func inflate(t *testing.T, s string) *AuthnRequest {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	r := flate.NewReader(bytes.NewReader(decoded))
	defer r.Close()

	var req AuthnRequest
	require.NoError(t, xml.NewDecoder(r).Decode(&req))
	return &req
}

type mockStore struct {
	session         *Session
	sessionLifetime time.Duration
}

func (s *mockStore) create(requestID, originalURL, metadata string, lifetimeSecs uint) error {
	s.session = &Session{OriginalURL: originalURL, Metadata: metadata}
	s.sessionLifetime = time.Duration(lifetimeSecs) * time.Second // nolint:gosec // dismiss G115
	return nil
}

func (s *mockStore) get(requestID string) (*Session, error) {
	if s.session == nil {
		return nil, fleet.NewAuthRequiredError("session not found")
	}
	return s.session, nil
}

func (s *mockStore) expire(requestID string) error {
	s.session = nil
	return nil
}
func (s *mockStore) Fullfill(requestID string) (*Session, *Metadata, error) {
	return s.session, &Metadata{}, nil
}
