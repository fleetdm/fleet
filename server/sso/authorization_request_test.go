package sso

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"net/url"
	"strings"
	"testing"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAuthorizationRequest(t *testing.T) {
	store := &mockStore{}

	metadata, err := xml.Marshal(&saml.EntityDescriptor{
		EntityID: "test",
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SingleSignOnServices: []saml.Endpoint{
					{Binding: saml.HTTPRedirectBinding, Location: "http://example.com"},
				},
			},
		},
	})
	require.NoError(t, err)

	samlProvider, err := SAMLProviderFromConfiguredMetadata(context.Background(),
		"issuer",
		"http://localhost:8001/api/v1/fleet/sso/callback",
		&fleet.SSOProviderSettings{
			IDPName:  "Fleet",
			Metadata: string(metadata),
		},
	)
	require.NoError(t, err)

	idpURL, err := CreateAuthorizationRequest(context.Background(),
		samlProvider,
		store,
		"/redir",
	)
	require.NoError(t, err)

	parsed, err := url.Parse(idpURL)
	require.NoError(t, err)
	assert.Equal(t, "example.com", parsed.Host)
	q := parsed.Query()
	encoded := q.Get("SAMLRequest")
	assert.NotEmpty(t, encoded)
	authReq := inflate(t, encoded)
	assert.Equal(t, "issuer", authReq.Issuer.Value)
	assert.Equal(t, "Fleet", authReq.ProviderName)
	assert.True(t, strings.HasPrefix(authReq.ID, "id"), authReq.ID)

	ssn := store.session
	require.NotNil(t, ssn)
	assert.Equal(t, "/redir", ssn.OriginalURL)

	var meta saml.EntityDescriptor
	require.NoError(t, xml.Unmarshal([]byte(ssn.Metadata), &meta))
	assert.Equal(t, "test", meta.EntityID)
}

func inflate(t *testing.T, s string) *saml.AuthnRequest {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	r := flate.NewReader(bytes.NewReader(decoded))
	defer r.Close()

	var req saml.AuthnRequest
	require.NoError(t, xml.NewDecoder(r).Decode(&req))
	return &req
}

type mockStore struct {
	session *Session
}

func (s *mockStore) create(relayStateToken, requestID, originalURL, metadata string, lifetimeSecs uint) error {
	s.session = &Session{
		RequestID:   requestID,
		OriginalURL: originalURL,
		Metadata:    metadata,
	}
	return nil
}

func (s *mockStore) get(relayStateToken string) (*Session, error) {
	if s.session == nil {
		return nil, fleet.NewAuthRequiredError("session not found")
	}
	return s.session, nil
}

func (s *mockStore) expire(relayStateToken string) error {
	s.session = nil
	return nil
}

func (s *mockStore) Fullfill(relayStateToken string) (*Session, error) {
	return s.session, nil
}
