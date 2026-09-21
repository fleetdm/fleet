package sso

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/fleet"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stretchr/testify/require"
)

type fakeSessionStore struct {
	sessions map[string]*Session
}

func (f *fakeSessionStore) create(sessionID, requestID, originalURL, metadata string, _ uint, requestData SSORequestData) error {
	f.sessions[sessionID] = &Session{RequestID: requestID, OriginalURL: originalURL, Metadata: metadata, RequestData: requestData}
	return nil
}

func (f *fakeSessionStore) get(sessionID string) (*Session, error) {
	sess, ok := f.sessions[sessionID]
	if !ok {
		return nil, &sessionNotFoundError{authRequired: fleet.NewAuthRequiredError("session not found")}
	}
	return sess, nil
}

func (f *fakeSessionStore) expire(sessionID string) error {
	delete(f.sessions, sessionID)
	return nil
}

func (f *fakeSessionStore) Fullfill(sessionID string) (*Session, error) {
	sess, err := f.get(sessionID)
	if err != nil {
		return nil, err
	}
	delete(f.sessions, sessionID)
	return sess, nil
}

func TestSAMLProviderFromSessionOrConfiguredMetadataRequestIDValidation(t *testing.T) {
	tm, err := time.Parse(time.UnixDate, "Sun Apr 30 22:09:50 UTC 2017")
	require.NoError(t, err)
	saml.TimeNow = func() time.Time { return tm }
	saml.Clock = dsig.NewFakeClockAt(tm)

	samlResponse, err := base64.StdEncoding.DecodeString(testResponse)
	require.NoError(t, err)

	const sessionID = "session-12345678"
	expectedAudiences := []string{"kolide"}

	for _, tc := range []struct {
		name             string
		idpLoginEnabled  bool
		withSession      bool
		sessionRequestID string
		wantNoSession    bool
		wantVerifyErr    string
	}{
		{
			name:             "SP-initiated, request ID matches, IdP-initiated login enabled",
			idpLoginEnabled:  true,
			withSession:      true,
			sessionRequestID: testRequestID,
		},
		{
			name:             "SP-initiated, request ID mismatch, IdP-initiated login enabled",
			idpLoginEnabled:  true,
			withSession:      true,
			sessionRequestID: "some-other-request-id",
			wantVerifyErr:    "`InResponseTo` does not match",
		},
		{
			name:             "SP-initiated, request ID mismatch, IdP-initiated login disabled",
			idpLoginEnabled:  false,
			withSession:      true,
			sessionRequestID: "some-other-request-id",
			wantVerifyErr:    "`InResponseTo` does not match",
		},
		{
			name:            "IdP-initiated, IdP-initiated login enabled",
			idpLoginEnabled: true,
		},
		{
			name:            "IdP-initiated, IdP-initiated login disabled",
			idpLoginEnabled: false,
			wantNoSession:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeSessionStore{sessions: map[string]*Session{}}
			var reqSessionID string
			if tc.withSession {
				reqSessionID = sessionID
				require.NoError(t, store.create(sessionID, tc.sessionRequestID, "/hosts", testSalesforceMetadataXML, 60, SSORequestData{}))
			}
			settings := &fleet.SSOSettings{
				EnableSSO:         true,
				EnableSSOIdPLogin: tc.idpLoginEnabled,
			}
			settings.EntityID = "https://fleet-dev-ed.my.salesforce.com"
			settings.Metadata = testSalesforceMetadataXML

			provider, requestID, redirectURL, err := SAMLProviderFromSessionOrConfiguredMetadata(
				context.Background(), reqSessionID, store, testACSURL, settings, expectedAudiences,
			)
			if tc.wantNoSession {
				require.ErrorIs(t, err, ErrSessionNotFound)
				return
			}
			require.NoError(t, err)
			if tc.withSession {
				require.Equal(t, tc.sessionRequestID, requestID)
				require.Equal(t, "/hosts", redirectURL)
				require.False(t, provider.AllowIDPInitiated)
			} else {
				require.Empty(t, requestID)
				require.Equal(t, "/", redirectURL)
				require.True(t, provider.AllowIDPInitiated)
			}

			auth, err := ParseAndVerifySAMLResponse(provider, samlResponse, requestID, testACSURL)
			if tc.wantVerifyErr != "" {
				require.ErrorContains(t, err, tc.wantVerifyErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "john@kolide.co", auth.UserID())
		})
	}
}
