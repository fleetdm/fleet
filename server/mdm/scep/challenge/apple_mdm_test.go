package challenge

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net"
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	scepmock "github.com/fleetdm/fleet/v4/server/mock/scep"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/require"
)

const testStaticChallenge = "static-challenge"

type fakeAppleSCEPStore struct {
	consume      func(challenge string) (*fleet.AppleSCEPChallengeInfo, error)
	consumed     []string
	setSerialErr error
	serials      map[string]int64
}

func (s *fakeAppleSCEPStore) ConsumeAppleSCEPChallenge(_ context.Context, challenge string) (*fleet.AppleSCEPChallengeInfo, error) {
	s.consumed = append(s.consumed, challenge)
	return s.consume(challenge)
}

func (s *fakeAppleSCEPStore) SetAppleSCEPChallengeIssuedCert(_ context.Context, challenge string, serial int64) error {
	if s.setSerialErr != nil {
		return s.setSerialErr
	}
	if s.serials == nil {
		s.serials = map[string]int64{}
	}
	s.serials[challenge] = serial
	return nil
}

func (s *fakeAppleSCEPStore) GetAllMDMConfigAssetsByName(_ context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
	return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetSCEPChallenge: {Name: fleet.MDMAssetSCEPChallenge, Value: []byte(testStaticChallenge)},
	}, nil
}

func (s *fakeAppleSCEPStore) GetABMTokenByOrgName(context.Context, string) (*fleet.ABMToken, error) {
	return nil, errors.New("not implemented")
}

func notFoundConsume(string) (*fleet.AppleSCEPChallengeInfo, error) {
	return nil, errors.New("not found")
}

func newTestSigner(t *testing.T) *depot.Signer {
	caCert, caKey, err := depot.NewSCEPCACertKey()
	require.NoError(t, err)
	var serial int64
	return depot.NewSigner(&scepmock.Depot{
		CAFunc: func([]byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
			return []*x509.Certificate{caCert}, caKey, nil
		},
		SerialFunc: func() (*big.Int, error) {
			serial++
			return big.NewInt(serial), nil
		},
		PutFunc: func(string, *x509.Certificate) error { return nil },
	})
}

// newCraftedCSR returns a CSR asking for everything Fleet must not copy: extra Subject attributes, every SAN
// type, and its own copy of the binding extension.
func newCraftedCSR(t *testing.T, ous ...string) *x509.CertificateRequest {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	forged, err := asn1.MarshalWithParams(`{"v":1,"purpose":"ade","udid":"forged"}`, "utf8")
	require.NoError(t, err)
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         "Not Fleet",
			Organization:       []string{"Evil Corp"},
			OrganizationalUnit: ous,
			Country:            []string{"US"},
			Locality:           []string{"Somewhere"},
		},
		DNSNames:        []string{"evil.example.com"},
		EmailAddresses:  []string{"evil@example.com"},
		IPAddresses:     []net.IP{net.ParseIP("10.0.0.1")},
		URIs:            []*url.URL{{Scheme: "urn", Opaque: "device:apple:uuid:forged"}},
		ExtraExtensions: []pkix.Extension{{Id: apple_mdm.AppleMDMCertificateBindingExtensionOID, Value: forged}},
	}, key)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(der)
	require.NoError(t, err)
	return csr
}

// requireFleetIdentityCert checks that only Fleet-controlled fields made it into the certificate, and returns the
// decoded binding extension, or nil if there is none.
func requireFleetIdentityCert(t *testing.T, cert *x509.Certificate, newEnrollment bool) map[string]any {
	t.Helper()
	require.Equal(t, "Fleet Identity", cert.Subject.CommonName)
	require.Equal(t, []string{"Fleet"}, cert.Subject.Organization)
	if newEnrollment {
		require.Equal(t, []string{apple_mdm.FleetEnrollmentSubjectOU}, cert.Subject.OrganizationalUnit)
	} else {
		require.Empty(t, cert.Subject.OrganizationalUnit)
	}
	require.Empty(t, cert.Subject.Country)
	require.Empty(t, cert.Subject.Locality)
	require.Empty(t, cert.DNSNames)
	require.Empty(t, cert.EmailAddresses)
	require.Empty(t, cert.IPAddresses)
	require.Empty(t, cert.URIs)

	var binding map[string]any
	for _, ext := range cert.Extensions {
		if !ext.Id.Equal(apple_mdm.AppleMDMCertificateBindingExtensionOID) {
			continue
		}
		require.Nil(t, binding, "binding extension present more than once")
		require.False(t, ext.Critical)
		var raw string
		rest, err := asn1.UnmarshalWithParams(ext.Value, &raw, "utf8")
		require.NoError(t, err)
		require.Empty(t, rest)
		require.NoError(t, json.Unmarshal([]byte(raw), &binding))
	}
	return binding
}

func TestAppleMDMChallengeMiddlewareStatic(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	t.Run("enabled and matching signs without binding", func(t *testing.T) {
		store := &fakeAppleSCEPStore{consume: notFoundConsume}
		mw := AppleMDMChallengeMiddleware(logger, store, true, newTestSigner(t))

		cert, err := mw(t.Context(), &scep.CSRReqMessage{
			ChallengePassword: testStaticChallenge,
			CSR:               newCraftedCSR(t, "extra-ou", apple_mdm.FleetEnrollmentSubjectOU),
		})
		require.NoError(t, err)
		require.Nil(t, requireFleetIdentityCert(t, cert, true))
		require.Empty(t, store.consumed)
	})

	t.Run("enabled and not matching falls through to dynamic", func(t *testing.T) {
		store := &fakeAppleSCEPStore{consume: notFoundConsume}
		mw := AppleMDMChallengeMiddleware(logger, store, true, newTestSigner(t))

		_, err := mw(t.Context(), &scep.CSRReqMessage{ChallengePassword: "other", CSR: newCraftedCSR(t)})
		require.Error(t, err)
		require.Equal(t, []string{"other"}, store.consumed)
	})

	t.Run("disabled rejects the static challenge", func(t *testing.T) {
		store := &fakeAppleSCEPStore{consume: notFoundConsume}
		mw := AppleMDMChallengeMiddleware(logger, store, false, newTestSigner(t))

		_, err := mw(t.Context(), &scep.CSRReqMessage{ChallengePassword: testStaticChallenge, CSR: newCraftedCSR(t)})
		require.Error(t, err)
		require.Equal(t, []string{testStaticChallenge}, store.consumed)
	})
}

func TestAppleMDMChallengeMiddlewareDynamic(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	info := func(purpose fleet.AppleMDMCertPurpose) *fleet.AppleSCEPChallengeInfo {
		return &fleet.AppleSCEPChallengeInfo{
			Purpose:        purpose,
			UUID:           new("host-uuid"),
			Serial:         new("hw-serial"),
			IDPAccountUUID: new("idp-uuid"),
		}
	}

	cases := []struct {
		purpose       fleet.AppleMDMCertPurpose
		newEnrollment bool
		want          map[string]any
	}{
		{fleet.AppleMDMCertPurposeADE, true, map[string]any{"v": 1.0, "purpose": "ade", "udid": "host-uuid", "serial": "hw-serial"}},
		{fleet.AppleMDMCertPurposeOTAPhaseOne, true, map[string]any{"v": 1.0, "purpose": "ota_phase1", "udid": "host-uuid", "serial": "hw-serial"}},
		{fleet.AppleMDMCertPurposeOTAPhaseTwo, true, map[string]any{"v": 1.0, "purpose": "ota_phase2", "udid": "host-uuid", "serial": "hw-serial"}},
		{fleet.AppleMDMCertPurposeADUE, true, map[string]any{"v": 1.0, "purpose": "adue", "idp_account_uuid": "idp-uuid"}},
		{fleet.AppleMDMCertPurposeSCEPRenewal, false, map[string]any{"v": 1.0, "purpose": "renewal", "enrollment_id": "host-uuid"}},
	}
	for _, c := range cases {
		t.Run(string(c.purpose), func(t *testing.T) {
			store := &fakeAppleSCEPStore{consume: func(string) (*fleet.AppleSCEPChallengeInfo, error) { return info(c.purpose), nil }}
			// static enabled too, to show a dynamic challenge still works alongside it
			mw := AppleMDMChallengeMiddleware(logger, store, true, newTestSigner(t))

			var ous []string
			if c.newEnrollment {
				ous = []string{apple_mdm.FleetEnrollmentSubjectOU}
			}
			cert, err := mw(t.Context(), &scep.CSRReqMessage{ChallengePassword: "dynamic", CSR: newCraftedCSR(t, ous...)})
			require.NoError(t, err)
			require.Equal(t, c.want, requireFleetIdentityCert(t, cert, c.newEnrollment))
			require.Equal(t, map[string]int64{"dynamic": cert.SerialNumber.Int64()}, store.serials)
		})
	}

	t.Run("unsupported purpose", func(t *testing.T) {
		store := &fakeAppleSCEPStore{consume: func(string) (*fleet.AppleSCEPChallengeInfo, error) {
			return info(fleet.AppleMDMCertPurposeACME), nil
		}}
		mw := AppleMDMChallengeMiddleware(logger, store, false, newTestSigner(t))

		_, err := mw(t.Context(), &scep.CSRReqMessage{ChallengePassword: "dynamic", CSR: newCraftedCSR(t)})
		require.Error(t, err)
	})

	t.Run("recording the serial fails", func(t *testing.T) {
		store := &fakeAppleSCEPStore{
			consume:      func(string) (*fleet.AppleSCEPChallengeInfo, error) { return info(fleet.AppleMDMCertPurposeADE), nil },
			setSerialErr: errors.New("db down"),
		}
		mw := AppleMDMChallengeMiddleware(logger, store, false, newTestSigner(t))

		cert, err := mw(t.Context(), &scep.CSRReqMessage{ChallengePassword: "dynamic", CSR: newCraftedCSR(t)})
		require.NoError(t, err)
		require.NotNil(t, cert)
	})
}
