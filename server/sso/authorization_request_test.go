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
	"time"

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

	sessionID, idpURL, err := CreateAuthorizationRequest(context.Background(),
		samlProvider,
		store,
		"/redir",
		0,
	)
	require.NoError(t, err)
	assert.Equal(t, 300*time.Second, store.sessionLifetime) // check default is used
	require.NotEmpty(t, sessionID)

	parsed, err := url.Parse(idpURL)
	require.NoError(t, err)
	assert.Equal(t, "example.com", parsed.Host)
	q := parsed.Query()
	encoded := q.Get("SAMLRequest")
	assert.NotEmpty(t, encoded)
	authReq := inflate(t, encoded)
	assert.Equal(t, "issuer", authReq.Issuer.Value)
	assert.Equal(t, "Fleet", authReq.ProviderName)
	assert.Equal(t, saml.EmailAddressNameIDFormat, authReq.NameIDPolicy)
	assert.True(t, strings.HasPrefix(authReq.ID, "id"), authReq.ID)

	ssn := store.session
	require.NotNil(t, ssn)
	assert.Equal(t, "/redir", ssn.OriginalURL)
	assert.Equal(t, 5*time.Minute, store.sessionLifetime)

	var meta saml.EntityDescriptor
	require.NoError(t, xml.Unmarshal([]byte(ssn.Metadata), &meta))
	assert.Equal(t, "test", meta.EntityID)

	sessionTTL := uint(3600) // seconds
	sessionID2, _, err := CreateAuthorizationRequest(context.Background(),
		samlProvider,
		store,
		"/redir",
		sessionTTL,
	)
	require.NoError(t, err)
	assert.Equal(t, 1*time.Hour, store.sessionLifetime)
	require.NotEmpty(t, sessionID2)
	require.NotEqual(t, sessionID, sessionID2)
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
	session         *Session
	sessionLifetime time.Duration
}

func (s *mockStore) create(sessionID, requestID, originalURL, metadata string, lifetimeSecs uint) error {
	s.session = &Session{
		RequestID:   requestID,
		OriginalURL: originalURL,
		Metadata:    metadata,
	}
	s.sessionLifetime = time.Duration(lifetimeSecs) * time.Second // nolint:gosec // dismiss G115
	return nil
}

func (s *mockStore) get(sessionID string) (*Session, error) {
	if s.session == nil {
		return nil, fleet.NewAuthRequiredError("session not found")
	}
	return s.session, nil
}

func (s *mockStore) expire(sessionID string) error {
	s.session = nil
	return nil
}

func (s *mockStore) Fullfill(sessionID string) (*Session, error) {
	return s.session, nil
}
