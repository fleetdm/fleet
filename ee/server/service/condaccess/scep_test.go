package condaccess

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"net"
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/mock"
	mockscep "github.com/fleetdm/fleet/v4/server/mock/scep"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeMiddleware(t *testing.T) {
	teamAID := uint(1)
	teamBID := uint(2)

	cases := []struct {
		name      string
		challenge string
		csrURIs   []string
		mdm       config.MDMConfig
		wantErr   string // empty when a certificate is issued
	}{
		{
			name:      "empty challenge is rejected",
			challenge: "",
			wantErr:   "missing challenge",
		},
		{
			name:      "request without a device UUID is rejected",
			challenge: "one-time",
			csrURIs:   []string{"https://example.com"},
			wantErr:   "no device UUID",
		},
		{
			name:      "request for an unknown host is rejected",
			challenge: "one-time",
			csrURIs:   []string{"urn:device:fleet:uuid:unknown"},
			wantErr:   "unknown host",
		},
		{
			name:      "host lookup failure is an error",
			challenge: "one-time",
			csrURIs:   []string{"urn:device:fleet:uuid:db-down"},
			wantErr:   "looking up host",
		},
		{
			name:      "unknown secret is rejected",
			challenge: "unknown-secret",
			csrURIs:   []string{"urn:device:fleet:uuid:host-1"},
			wantErr:   "invalid challenge",
		},
		{
			name:      "team-scoped secret is rejected",
			challenge: "secret-team-a",
			csrURIs:   []string{"urn:device:apple:uuid:host-1"},
			wantErr:   "invalid challenge",
		},
		{
			name:      "different team-scoped secret is also rejected",
			challenge: "secret-team-b",
			csrURIs:   []string{"urn:device:apple:uuid:host-1"},
			wantErr:   "invalid challenge",
		},
		{
			name:      "global secret is accepted",
			challenge: "global-secret",
			csrURIs:   []string{"urn:device:apple:uuid:host-1"},
		},
		{
			name:      "global secret is refused for a Mac when Apple hosts get one-time enroll secrets",
			challenge: "global-secret",
			csrURIs:   []string{"urn:device:apple:uuid:mac-1"},
			mdm:       config.MDMConfig{AppleOneTimeEnrollSecrets: true},
			wantErr:   "invalid challenge",
		},
		{
			name:      "global secret still works for a Mac when only Windows hosts get one-time enroll secrets",
			challenge: "global-secret",
			csrURIs:   []string{"urn:device:apple:uuid:mac-1"},
			mdm:       config.MDMConfig{WindowsOneTimeEnrollSecrets: true},
		},
		{
			name:      "global secret is refused for a Windows host when Windows hosts get one-time enroll secrets",
			challenge: "global-secret",
			csrURIs:   []string{"urn:device:fleet:uuid:win-1"},
			mdm:       config.MDMConfig{WindowsOneTimeEnrollSecrets: true},
			wantErr:   "invalid challenge",
		},
		{
			name:      "global secret still works for a Windows host when only Apple hosts get one-time enroll secrets",
			challenge: "global-secret",
			csrURIs:   []string{"urn:device:fleet:uuid:win-1"},
			mdm:       config.MDMConfig{AppleOneTimeEnrollSecrets: true},
		},
		{
			name:      "one-time challenge for the requesting host is accepted",
			challenge: "one-time",
			csrURIs:   []string{"urn:device:fleet:uuid:host-1"},
		},
		{
			name:      "one-time challenge is accepted when one-time enroll secrets are on",
			challenge: "one-time",
			csrURIs:   []string{"urn:device:fleet:uuid:host-1"},
			mdm:       config.MDMConfig{AppleOneTimeEnrollSecrets: true, WindowsOneTimeEnrollSecrets: true},
		},
		{
			name:      "one-time challenge for another host is rejected",
			challenge: "one-time",
			csrURIs:   []string{"urn:device:fleet:uuid:host-2"},
			wantErr:   "invalid challenge",
		},
		{
			name:      "one-time challenge is checked against the first device URI, the one the certificate gets",
			challenge: "one-time",
			csrURIs:   []string{"urn:device:apple:uuid:host-2", "urn:device:apple:uuid:host-1"},
			wantErr:   "invalid challenge",
		},
		{
			name:      "one-time challenge lookup failure is an error",
			challenge: "db-down",
			csrURIs:   []string{"urn:device:fleet:uuid:host-1"},
			wantErr:   "consuming conditional access SCEP challenge",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.DataStore)
			ds.VerifyEnrollSecretFunc = func(_ context.Context, secret string) (*fleet.EnrollSecret, error) {
				switch secret {
				case "secret-team-a":
					return &fleet.EnrollSecret{Secret: secret, TeamID: &teamAID}, nil
				case "secret-team-b":
					return &fleet.EnrollSecret{Secret: secret, TeamID: &teamBID}, nil
				case "global-secret":
					return &fleet.EnrollSecret{Secret: secret, TeamID: nil}, nil
				default:
					return nil, common_mysql.NotFound("enroll_secret")
				}
			}
			ds.HostByUUIDFunc = func(_ context.Context, hostUUID string) (*fleet.Host, error) {
				switch hostUUID {
				case "mac-1", "host-1", "host-2":
					return &fleet.Host{UUID: hostUUID, Platform: "darwin"}, nil
				case "win-1":
					return &fleet.Host{UUID: hostUUID, Platform: "windows"}, nil
				case "db-down":
					return nil, errors.New("db down")
				default:
					return nil, common_mysql.NotFound("Host")
				}
			}
			ds.ConsumeConditionalAccessSCEPChallengeFunc = func(_ context.Context, challenge, hostUUID string) error {
				switch {
				case challenge == "one-time" && hostUUID == "host-1":
					return nil
				case challenge == "db-down":
					return errors.New("db down")
				default:
					return common_mysql.NotFound("conditional access SCEP challenge")
				}
			}
			signer := callbackSignerFunc(func(*x509.CertificateRequest, pkix.Name, func(*x509.Certificate)) (*x509.Certificate, error) {
				return &x509.Certificate{}, nil
			})

			csr := &x509.CertificateRequest{}
			for _, s := range tc.csrURIs {
				u, err := url.Parse(s)
				require.NoError(t, err)
				csr.URIs = append(csr.URIs, u)
			}

			cert, err := challengeMiddleware(ds, slog.New(slog.DiscardHandler), tc.mdm, signer).SignCSRContext(t.Context(), &scep.CSRReqMessage{
				ChallengePassword: tc.challenge,
				CSR:               csr,
			})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, cert)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cert)
		})
	}
}

type callbackSignerFunc func(csr *x509.CertificateRequest, subject pkix.Name, callback func(*x509.Certificate)) (*x509.Certificate, error)

func (f callbackSignerFunc) SignX509CSRWithCallback(csr *x509.CertificateRequest, subject pkix.Name, callback func(*x509.Certificate)) (*x509.Certificate, error) {
	return f(csr, subject, callback)
}

func TestChallengeMiddlewareSignsOnlyTheDeviceURI(t *testing.T) {
	caCert, caKey, err := scepdepot.NewSCEPCACertKey()
	require.NoError(t, err)
	depot := &mockscep.Depot{}
	depot.CAFunc = func([]byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
		return []*x509.Certificate{caCert}, caKey, nil
	}
	depot.SerialFunc = func() (*big.Int, error) { return big.NewInt(1), nil }
	depot.PutFunc = func(string, *x509.Certificate) error { return nil }
	ds := new(mock.DataStore)
	ds.HostByUUIDFunc = func(_ context.Context, hostUUID string) (*fleet.Host, error) {
		return &fleet.Host{UUID: hostUUID, Platform: "darwin"}, nil
	}
	ds.VerifyEnrollSecretFunc = func(context.Context, string) (*fleet.EnrollSecret, error) {
		return nil, common_mysql.NotFound("enroll_secret")
	}
	ds.ConsumeConditionalAccessSCEPChallengeFunc = func(_ context.Context, _, hostUUID string) error {
		if hostUUID != "host-1" {
			return common_mysql.NotFound("conditional access SCEP challenge")
		}
		return nil
	}
	signer := challengeMiddleware(ds, slog.New(slog.DiscardHandler), config.MDMConfig{}, scepdepot.NewSigner(depot))

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.CertificateRequest{
		Subject:        pkix.Name{CommonName: fleet.ConditionalAccessOktaCertificateCN, OrganizationalUnit: []string{"fleet-renewal"}},
		DNSNames:       []string{"evil.example.com"},
		EmailAddresses: []string{"attacker@example.com"},
		IPAddresses:    []net.IP{net.ParseIP("10.0.0.1")},
	}
	for _, s := range []string{"https://evil.example.com", "urn:device:apple:uuid:host-1", "urn:device:fleet:uuid:host-2"} {
		u, err := url.Parse(s)
		require.NoError(t, err)
		tmpl.URIs = append(tmpl.URIs, u)
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, tmpl, key)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(der)
	require.NoError(t, err)

	cert, err := signer.SignCSRContext(t.Context(), &scep.CSRReqMessage{CSR: csr, ChallengePassword: "one-time"})
	require.NoError(t, err)
	require.Len(t, cert.URIs, 1)
	require.Equal(t, "urn:device:apple:uuid:host-1", cert.URIs[0].String())
	require.Empty(t, cert.DNSNames)
	require.Empty(t, cert.EmailAddresses)
	require.Empty(t, cert.IPAddresses)
	// The renewal ID rides in the OU, so the subject is kept.
	require.Equal(t, []string{"fleet-renewal"}, cert.Subject.OrganizationalUnit)
	require.Equal(t, fleet.ConditionalAccessOktaCertificateCN, cert.Subject.CommonName)
}

func TestPKIOperationUndecryptableEnvelope(t *testing.T) {
	caCert, caKey, err := scepdepot.NewSCEPCACertKey()
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	ds := new(mock.DataStore)
	ds.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetConditionalAccessCACert: {Name: fleet.MDMAssetConditionalAccessCACert, Value: certPEM},
			fleet.MDMAssetConditionalAccessCAKey:  {Name: fleet.MDMAssetConditionalAccessCAKey, Value: keyPEM},
		}, nil
	}

	signerCalled := false
	signer := scepserver.CSRSignerContextFunc(func(context.Context, *scep.CSRReqMessage) (*x509.Certificate, error) {
		signerCalled = true
		return &x509.Certificate{}, nil
	})
	svc := NewSCEPService(ds, signer, slog.New(slog.DiscardHandler))

	req, err := mdmtest.NewPKCSReqUndecryptableBy(caCert)
	require.NoError(t, err)

	respBytes, err := svc.PKIOperation(t.Context(), req.Raw)
	require.NoError(t, err)

	certRep, err := scep.ParsePKIMessage(respBytes)
	require.NoError(t, err)
	assert.Equal(t, scep.FAILURE, certRep.PKIStatus)
	assert.Equal(t, scep.BadRequest, certRep.FailInfo)
	assert.Equal(t, req.TransactionID, certRep.TransactionID)
	assert.False(t, signerCalled, "signer must not run when the envelope cannot be decrypted")
}

func TestPKIOperationRejectsDeeplyNestedBER(t *testing.T) {
	svc := NewSCEPService(nil, nil, slog.New(slog.DiscardHandler))

	deep := bytes.Repeat([]byte{0x30, 0x80}, cryptoutil.MaxBERDepth+1)
	_, err := svc.PKIOperation(t.Context(), deep)

	var badReq *fleet.BadRequestError
	require.ErrorAs(t, err, &badReq)
	require.ErrorIs(t, err, cryptoutil.ErrBERTooDeep)
}
