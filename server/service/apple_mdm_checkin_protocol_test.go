package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nano_service "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const testGetTokenUDID = "ABC-DEF"

// newGetTokenTestService returns a GetToken service backed by mocks that resolve to a single
// default ABM token and an unknown host. Tests override the funcs they care about.
func newGetTokenTestService(t *testing.T) (*MDMAppleGetTokenService, *mock.Store, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	ds := new(mock.Store)
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: 1, ServerUUID: "default-server-uuid", IsDefault: true}}, nil
	}
	ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
		return nil, newNotFoundError()
	}
	ds.GetHostDEPAssignmentFunc = func(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
		return nil, newNotFoundError()
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{fleet.MDMAssetABMKey: {Name: fleet.MDMAssetABMKey, Value: keyPEM}}, nil
	}

	svc := NewMDMAppleGetTokenService(ds, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return svc, ds, key
}

func getTokenRequest(t *testing.T, svc *MDMAppleGetTokenService, serviceType string) (*mdm.GetTokenResponse, error) {
	t.Helper()
	return svc.GetToken(
		&mdm.Request{Context: t.Context()},
		&mdm.GetToken{UDID: testGetTokenUDID, TokenServiceType: serviceType},
	)
}

func TestMDMAppleGetTokenClaims(t *testing.T) {
	svc, _, key := newGetTokenTestService(t)

	resp, err := getTokenRequest(t, svc, fleet.TokenServiceTypeMAID)
	require.NoError(t, err)

	var claims appleMAIDTokenClaims
	_, err = jwt.ParseWithClaims(string(resp.TokenData), &claims, func(*jwt.Token) (any, error) { return &key.PublicKey, nil })
	require.NoError(t, err)
	require.Equal(t, fleet.TokenServiceTypeMAID, claims.TokenServiceType)
	require.Equal(t, "default-server-uuid", claims.Issuer)
	require.NotEmpty(t, claims.ID)
	require.NotNil(t, claims.IssuedAt)
}

func TestMDMAppleGetTokenSigningTokenSelection(t *testing.T) {
	depAssignedHost := func(ds *mock.Store, abmTokenID *uint) {
		ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
			return &fleet.HostLite{ID: 42, UUID: identifier}, nil
		}
		ds.GetHostDEPAssignmentFunc = func(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
			return &fleet.HostDEPAssignment{HostID: hostID, ABMTokenID: abmTokenID}, nil
		}
	}

	cases := []struct {
		name        string
		serviceType string
		setup       func(ds *mock.Store)
		wantIssuer  string
	}{
		{
			name:        "unsupported token service type",
			serviceType: "com.apple.watch.pairing",
		},
		{
			name:        "unknown host falls back to the default token",
			serviceType: fleet.TokenServiceTypeMAID,
			wantIssuer:  "default-server-uuid",
		},
		{
			name:        "DEP assigned host uses its own token",
			serviceType: fleet.TokenServiceTypeMAID,
			setup: func(ds *mock.Store) {
				depAssignedHost(ds, new(uint(2)))
				ds.GetABMTokenByIDFunc = func(ctx context.Context, tokenID uint) (*fleet.ABMToken, error) {
					return &fleet.ABMToken{ID: tokenID, ServerUUID: "dep-server-uuid"}, nil
				}
			},
			wantIssuer: "dep-server-uuid",
		},
		{
			name:        "DEP assigned host does not fall back to the default token",
			serviceType: fleet.TokenServiceTypeMAID,
			setup: func(ds *mock.Store) {
				depAssignedHost(ds, nil)
			},
		},
		{
			name:        "no ABM tokens",
			serviceType: fleet.TokenServiceTypeMAID,
			setup: func(ds *mock.Store) {
				ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) { return nil, nil }
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, ds, key := newGetTokenTestService(t)
			if c.setup != nil {
				c.setup(ds)
			}

			resp, err := getTokenRequest(t, svc, c.serviceType)
			if c.wantIssuer == "" {
				require.ErrorContains(t, err, "HTTP status 400 (Bad Request)")
				return
			}

			require.NoError(t, err)
			var claims appleMAIDTokenClaims
			_, err = jwt.ParseWithClaims(string(resp.TokenData), &claims, func(*jwt.Token) (any, error) { return &key.PublicKey, nil })
			require.NoError(t, err)
			require.Equal(t, c.wantIssuer, claims.Issuer)
		})
	}
}

type recordingCheckinService struct {
	nano_service.CheckinAndCommandService
	authenticated bool
}

func (s *recordingCheckinService) Authenticate(*mdm.Request, *mdm.Authenticate) error {
	s.authenticated = true
	return nil
}

type fakeCertAssocStore struct {
	fleet.MDMAppleStore
	associated bool
	checkedID  string
}

func (s *fakeCertAssocStore) IsCertHashAssociated(r *mdm.Request, _ string) (bool, error) {
	s.checkedID = r.ID
	return s.associated, nil
}

// newBindingCert returns a certificate carrying binding, or no binding extension if it's nil.
func newBindingCert(t *testing.T, binding *apple_mdm.AppleMDMCertificateBindingExtension, ous ...string) *x509.Certificate {
	t.Helper()
	if binding == nil {
		return newCertWithExtensions(t, ous)
	}
	ext, err := apple_mdm.BuildAppleMDMCertificateBindingExtension(*binding)
	require.NoError(t, err)
	return newCertWithExtensions(t, ous, ext)
}

// newRawBindingCert returns a certificate whose binding extension holds value as is, for values the builder can't
// produce.
func newRawBindingCert(t *testing.T, value []byte) *x509.Certificate {
	t.Helper()
	return newCertWithExtensions(t, nil, pkix.Extension{Id: apple_mdm.AppleMDMCertificateBindingExtensionOID, Value: value})
}

func newCertWithExtensions(t *testing.T, ous []string, exts ...pkix.Extension) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:    big.NewInt(1),
		Subject:         pkix.Name{CommonName: "Fleet Identity", OrganizationalUnit: ous},
		NotBefore:       time.Now().Add(-time.Hour),
		NotAfter:        time.Now().Add(time.Hour),
		ExtraExtensions: exts,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func TestCertVerifierEnrollmentCheckinServiceAuthenticate(t *testing.T) {
	const (
		udid         = "device-udid"
		serial       = "device-serial"
		enrollmentID = "adue-enrollment-id"
		idpAccount   = "idp-account-uuid"
		adueToken    = "adue-token"
	)
	device := func(udid, serial string) *mdm.Authenticate {
		return &mdm.Authenticate{UDID: udid, SerialNumber: serial}
	}
	userEnrollment := func(id string) *mdm.Authenticate {
		return &mdm.Authenticate{EnrollmentID: id}
	}
	bind := func(purpose fleet.AppleMDMCertPurpose, fields func(*apple_mdm.AppleMDMCertificateBindingExtension)) *apple_mdm.AppleMDMCertificateBindingExtension {
		b := &apple_mdm.AppleMDMCertificateBindingExtension{Purpose: purpose}
		if fields != nil {
			fields(b)
		}
		return b
	}
	deviceIdentity := func(b *apple_mdm.AppleMDMCertificateBindingExtension) { b.UDID, b.Serial = new(udid), new(serial) }
	utf8Value := func(s string) []byte {
		v, err := asn1.MarshalWithParams(s, "utf8")
		require.NoError(t, err)
		return v
	}
	validBindingJSON := `{"v":1,"purpose":"ade","udid":"` + udid + `","serial":"` + serial + `"}`

	cases := []struct {
		name          string
		cert          *x509.Certificate
		msg           *mdm.Authenticate
		authorization string
		static        bool
		associated    bool
		enrolled      bool
		allowed       bool
	}{
		{name: "ade matching", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADE, deviceIdentity)), msg: device(udid, serial), allowed: true},
		{name: "ade other udid", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADE, deviceIdentity)), msg: device("other", serial)},
		{name: "ade other serial", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADE, deviceIdentity)), msg: device(udid, "other")},
		{name: "ade as user enrollment", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADE, deviceIdentity)), msg: userEnrollment(enrollmentID)},
		{name: "ade with empty bound serial", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.UDID, b.Serial = new(udid), new("")
		})), msg: device(udid, "")},
		{name: "ota_phase2 matching", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeOTAPhaseTwo, deviceIdentity)), msg: device(udid, serial), allowed: true},
		{name: "ota_phase2 other udid", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeOTAPhaseTwo, deviceIdentity)), msg: device("other", serial)},
		{name: "ota_phase1 always rejected", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeOTAPhaseOne, deviceIdentity)), msg: device(udid, serial)},

		{name: "acme matching serial", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeACME, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.Serial = new(serial)
		})), msg: device(udid, serial), allowed: true},
		{name: "acme other serial", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeACME, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.Serial = new(serial)
		})), msg: device(udid, "other")},
		{name: "acme_renewal matching", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeACMERenewal, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.Serial, b.EnrollmentID = new(serial), new(udid)
		})), msg: device(udid, serial), allowed: true},
		{name: "acme_renewal other enrollment with matching serial", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeACMERenewal, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.Serial, b.EnrollmentID = new(serial), new(udid)
		})), msg: device("other", serial)},

		{name: "renewal of its own device", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeSCEPRenewal, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.EnrollmentID = new(udid)
		})), msg: device(udid, serial), allowed: true},
		{name: "renewal presented as another device", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeSCEPRenewal, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.EnrollmentID = new(udid)
		})), msg: device("host-b", serial)},
		{name: "renewal of a user enrollment", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeSCEPRenewal, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.EnrollmentID = new(enrollmentID)
		})), msg: userEnrollment(enrollmentID), allowed: true},

		{name: "adue matching", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADUE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.IDPAccountUUID = new(idpAccount)
		})), msg: userEnrollment(enrollmentID), authorization: "Bearer " + adueToken, allowed: true},
		{name: "adue existing enrollment", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADUE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.IDPAccountUUID = new(idpAccount)
		})), msg: userEnrollment(enrollmentID), authorization: "Bearer " + adueToken, enrolled: true},
		{name: "adue as device enrollment", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADUE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.IDPAccountUUID = new(idpAccount)
		})), msg: device(udid, serial), authorization: "Bearer " + adueToken},
		{name: "adue other idp account", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADUE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.IDPAccountUUID = new("other-account")
		})), msg: userEnrollment(enrollmentID), authorization: "Bearer " + adueToken},
		{name: "adue unknown token", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADUE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.IDPAccountUUID = new(idpAccount)
		})), msg: userEnrollment(enrollmentID), authorization: "Bearer unknown"},
		{name: "adue no bearer token", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADUE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.IDPAccountUUID = new(idpAccount)
		})), msg: userEnrollment(enrollmentID)},
		{name: "adue resend of associated cert", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADUE, func(b *apple_mdm.AppleMDMCertificateBindingExtension) {
			b.IDPAccountUUID = new(idpAccount)
		})), msg: userEnrollment(enrollmentID), associated: true, enrolled: true, allowed: true},

		{name: "unknown purpose", cert: newBindingCert(t, bind("future", deviceIdentity)), msg: device(udid, serial)},

		// a binding that doesn't parse is rejected even when the identity it claims would match, and static on doesn't
		// treat it as unbound
		{name: "raw JSON instead of DER", cert: newRawBindingCert(t, []byte(validBindingJSON)), msg: device(udid, serial), static: true},
		{name: "malformed JSON", cert: newRawBindingCert(t, utf8Value(`{"v":1,"purpose":`)), msg: device(udid, serial), static: true},
		{name: "trailing bytes after the binding", cert: newRawBindingCert(t, append(utf8Value(validBindingJSON), 0x00)), msg: device(udid, serial), static: true},
		{name: "missing version", cert: newRawBindingCert(t, utf8Value(`{"purpose":"ade","udid":"`+udid+`","serial":"`+serial+`"}`)), msg: device(udid, serial), static: true},
		{name: "unsupported version", cert: newRawBindingCert(t, utf8Value(`{"v":2,"purpose":"ade","udid":"`+udid+`","serial":"`+serial+`"}`)), msg: device(udid, serial), static: true},
		{name: "unsupported version but associated", cert: newRawBindingCert(t, utf8Value(`{"v":2,"purpose":"ade","udid":"`+udid+`","serial":"`+serial+`"}`)), msg: device(udid, serial), associated: true, allowed: true},
		{name: "supported version control", cert: newRawBindingCert(t, utf8Value(validBindingJSON)), msg: device(udid, serial), allowed: true},
		{name: "mismatched but associated", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADE, deviceIdentity)), msg: device("other", serial), associated: true, allowed: true},

		{name: "unbound with static on", cert: newBindingCert(t, nil), msg: device(udid, serial), static: true, allowed: true},
		{name: "unbound with static off", cert: newBindingCert(t, nil), msg: device(udid, serial)},
		{name: "unbound associated with static off", cert: newBindingCert(t, nil), msg: device(udid, serial), associated: true, allowed: true},
		{name: "bound with static on is still checked", cert: newBindingCert(t, bind(fleet.AppleMDMCertPurposeADE, deviceIdentity)), msg: device("other", serial), static: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, id string) (*fleet.NanoEnrollment, error) {
				if c.enrolled {
					return &fleet.NanoEnrollment{ID: id}, nil
				}
				return nil, nil
			}
			ds.GetADUEEnrollmentChallengeFunc = func(ctx context.Context, challenge string) (*fleet.ADUEEnrollmentChallenge, error) {
				if challenge != adueToken {
					return nil, newNotFoundError()
				}
				return &fleet.ADUEEnrollmentChallenge{IdPAccountUUID: idpAccount}, nil
			}
			store := &fakeCertAssocStore{associated: c.associated}
			next := &recordingCheckinService{}
			svc := newCertVerifierEnrollmentCheckinService(next, ds, store, config.MDMConfig{AppleSCEPStaticChallengeEnabled: c.static}, slog.New(slog.DiscardHandler))

			r := &mdm.Request{Context: t.Context(), Certificate: c.cert, Authorization: c.authorization}
			err := svc.Authenticate(r, c.msg)

			require.Equal(t, c.msg.Resolved().DeviceChannelID, store.checkedID)
			if c.allowed {
				require.NoError(t, err)
				require.True(t, next.authenticated)
				return
			}
			var statusErr *nano_service.HTTPStatusError
			require.ErrorAs(t, err, &statusErr)
			require.Equal(t, http.StatusForbidden, statusErr.Status)
			require.False(t, next.authenticated)
		})
	}

	t.Run("missing certificate is left to certauth", func(t *testing.T) {
		next := &recordingCheckinService{}
		svc := newCertVerifierEnrollmentCheckinService(next, new(mock.Store), &fakeCertAssocStore{}, config.MDMConfig{}, slog.New(slog.DiscardHandler))
		require.NoError(t, svc.Authenticate(&mdm.Request{Context: t.Context()}, device(udid, serial)))
		require.True(t, next.authenticated)
	})

	t.Run("no enrollment identifiers", func(t *testing.T) {
		next := &recordingCheckinService{}
		svc := newCertVerifierEnrollmentCheckinService(next, new(mock.Store), &fakeCertAssocStore{}, config.MDMConfig{}, slog.New(slog.DiscardHandler))
		err := svc.Authenticate(&mdm.Request{Context: t.Context(), Certificate: newBindingCert(t, nil)}, &mdm.Authenticate{})
		require.Error(t, err)
		require.False(t, next.authenticated)
	})
}
