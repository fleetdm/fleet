package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	ejbca_mock "github.com/fleetdm/fleet/v4/server/mock/ejbca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

// errStopBeforeActivity makes the datastore mock end a create or update before it records an activity, which the unit tests can't.
var errStopBeforeActivity = errors.New("mock error to avoid NewActivity panic")

const ejbcaP12Password = "p12-password"

// ejbcaClientP12 returns a base64 PKCS#12 of a self-signed client certificate valid between the times.
func ejbcaClientP12(t *testing.T, notBefore, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Fleet"}, NotBefore: notBefore, NotAfter: notAfter}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	p12, err := pkcs12.Modern.Encode(key, cert, nil, ejbcaP12Password)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(p12)
}

func validEJBCAClientP12(t *testing.T) string {
	return ejbcaClientP12(t, time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))
}

type ejbcaTestEnv struct {
	svc      *Service
	ctx      context.Context
	ejbca    *ejbca_mock.Service
	ds       *mock.Store
	created  *fleet.CertificateAuthority // what was sent to the datastore to create
	updated  *fleet.CertificateAuthority // what was sent to the datastore to update
	verified []fleet.EJBCA               // what EJBCA was asked to verify
}

func newEJBCATestEnv(t *testing.T, stored *fleet.CertificateAuthority) *ejbcaTestEnv {
	t.Helper()
	env := &ejbcaTestEnv{ds: new(mock.Store), ejbca: new(ejbca_mock.Service)}
	env.ejbca.VerifyConnectionFunc = func(_ context.Context, config fleet.EJBCA) error {
		env.verified = append(env.verified, config)
		return nil
	}
	env.ds.NewCertificateAuthorityFunc = func(_ context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
		env.created = ca
		return nil, errStopBeforeActivity
	}
	env.ds.GetCertificateAuthorityByIDFunc = func(_ context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
		require.True(t, includeSecrets, "an update merges with the stored credential")
		return stored, nil
	}
	env.ds.UpdateCertificateAuthorityByIDFunc = func(_ context.Context, _ uint, ca *fleet.CertificateAuthority) error {
		env.updated = ca
		return errStopBeforeActivity
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	env.svc = &Service{
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		ds:           env.ds,
		authz:        authorizer,
		ejbcaService: env.ejbca,
	}
	env.svc.config.Server.PrivateKey = "supersecret"
	env.ctx = viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	return env
}

func validEJBCA(t *testing.T) fleet.EJBCA {
	return fleet.EJBCA{
		Name:                          "EJBCA_WIFI",
		URL:                           "https://ejbca.example.com:8443",
		ClientP12:                     validEJBCAClientP12(t),
		Password:                      ejbcaP12Password,
		CertificateAuthorityName:      "ManagementCA",
		CertificateProfileName:        "fleetCertProfile",
		EndEntityProfileName:          "fleetEEProfile",
		CertificateCommonName:         "$FLEET_VAR_HOST_END_USER_EMAIL_IDP",
		UsernameTemplate:              "$FLEET_VAR_HOST_HARDWARE_SERIAL",
		CertificateUserPrincipalNames: []string{"$FLEET_VAR_HOST_END_USER_EMAIL_IDP"},
		TrustCABundlePEM:              "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
	}
}

func TestNewEJBCACertificateAuthority(t *testing.T) {
	t.Parallel()

	t.Run("creates the CA from the PKCS#12 and verifies the connection", func(t *testing.T) {
		t.Parallel()
		env := newEJBCATestEnv(t, nil)
		payload := validEJBCA(t)

		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
		require.ErrorIs(t, err, errStopBeforeActivity)

		ca := env.created
		require.NotNil(t, ca)
		assert.Equal(t, string(fleet.CATypeEJBCA), ca.Type)
		assert.Equal(t, payload.Name, *ca.Name)
		assert.Equal(t, payload.URL, *ca.URL)
		assert.Equal(t, payload.TrustCABundlePEM, *ca.TrustCABundlePEM)
		assert.Equal(t, payload.CertificateAuthorityName, *ca.EJBCACAName)
		assert.Equal(t, payload.CertificateProfileName, *ca.EJBCACertificateProfileName)
		assert.Equal(t, payload.EndEntityProfileName, *ca.EJBCAEndEntityProfileName)
		assert.Equal(t, payload.CertificateCommonName, *ca.CertificateCommonName)
		assert.Equal(t, payload.UsernameTemplate, *ca.EJBCAUsernameTemplate)
		assert.Equal(t, payload.CertificateUserPrincipalNames, *ca.CertificateUserPrincipalNames)
		assert.Nil(t, ca.Password, "the PKCS#12 password is not stored")

		// The certificate and the key come out of the PKCS#12, and the password is not among the fields that are stored.
		require.NotNil(t, ca.MTLSCertPEM)
		require.NotNil(t, ca.MTLSKeyPEM)
		leaf, err := certificate.FirstCertificateFromPEM([]byte(*ca.MTLSCertPEM))
		require.NoError(t, err)
		assert.Equal(t, "Fleet", leaf.Subject.CommonName)
		assert.Contains(t, *ca.MTLSKeyPEM, "PRIVATE KEY")

		require.Len(t, env.verified, 1)
		assert.Equal(t, *ca.MTLSCertPEM, env.verified[0].MTLSCertPEM)
		assert.Equal(t, *ca.MTLSKeyPEM, env.verified[0].MTLSKeyPEM)
		assert.Equal(t, payload.CertificateProfileName, env.verified[0].CertificateProfileName)
	})

	t.Run("optional fields are left unset", func(t *testing.T) {
		t.Parallel()
		env := newEJBCATestEnv(t, nil)
		payload := validEJBCA(t)
		payload.TrustCABundlePEM, payload.UsernameTemplate, payload.CertificateUserPrincipalNames = "", "", nil

		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
		require.ErrorIs(t, err, errStopBeforeActivity)
		assert.Nil(t, env.created.TrustCABundlePEM)
		assert.Nil(t, env.created.EJBCAUsernameTemplate)
		assert.Nil(t, env.created.CertificateUserPrincipalNames)
		assert.Equal(t, payload.CertificateCommonName, env.created.EffectiveEJBCAUsernameTemplate())
	})

	tests := []struct {
		name    string
		modify  func(t *testing.T, p *fleet.EJBCA)
		wantErr string
	}{
		{name: "invalid name", modify: func(_ *testing.T, p *fleet.EJBCA) { p.Name = "bad name" }, wantErr: "Invalid characters in the \"name\" field"},
		{name: "invalid URL", modify: func(_ *testing.T, p *fleet.EJBCA) { p.URL = "not a url" }, wantErr: "Invalid EJBCA URL"},
		{name: "no CA name", modify: func(_ *testing.T, p *fleet.EJBCA) { p.CertificateAuthorityName = "" }, wantErr: "EJBCA CA name cannot be empty"},
		{
			name:    "no certificate profile name",
			modify:  func(_ *testing.T, p *fleet.EJBCA) { p.CertificateProfileName = " " },
			wantErr: "certificate profile name cannot be empty",
		},
		{
			name:    "no end entity profile name",
			modify:  func(_ *testing.T, p *fleet.EJBCA) { p.EndEntityProfileName = "" },
			wantErr: "end entity profile name cannot be empty",
		},
		{name: "no common name", modify: func(_ *testing.T, p *fleet.EJBCA) { p.CertificateCommonName = "" }, wantErr: "common name cannot be empty"},
		{
			name:    "name too long",
			modify:  func(_ *testing.T, p *fleet.EJBCA) { p.CertificateProfileName = strings.Repeat("a", 256) },
			wantErr: "certificate profile name cannot be longer than 255 characters",
		},
		{
			name: "unsupported variable in the common name",
			modify: func(_ *testing.T, p *fleet.EJBCA) {
				p.CertificateCommonName = "$FLEET_VAR_HOST_END_USER_IDP_USERNAME_X"
			},
			wantErr: "FLEET_VAR_HOST_END_USER_IDP_USERNAME_X is not allowed in the common name",
		},
		{
			name:    "unsupported variable in the username template",
			modify:  func(_ *testing.T, p *fleet.EJBCA) { p.UsernameTemplate = "$FLEET_VAR_DIGICERT_DATA_X" },
			wantErr: "FLEET_VAR_DIGICERT_DATA_X is not allowed in the username template",
		},
		{
			name: "more than one UPN",
			modify: func(_ *testing.T, p *fleet.EJBCA) {
				p.CertificateUserPrincipalNames = []string{"a@example.com", "b@example.com"}
			},
			wantErr: "only one item can be added to certificate_user_principal_names",
		},
		{
			name:    "blank UPN",
			modify:  func(_ *testing.T, p *fleet.EJBCA) { p.CertificateUserPrincipalNames = []string{" "} },
			wantErr: "user principal name cannot be empty",
		},
		{
			name: "unsupported variable in the UPN",
			modify: func(_ *testing.T, p *fleet.EJBCA) {
				p.CertificateUserPrincipalNames = []string{"$FLEET_VAR_NDES_SCEP_CHALLENGE"}
			},
			wantErr: "FLEET_VAR_NDES_SCEP_CHALLENGE is not allowed in the user principal name",
		},
		{
			name:    "trust bundle too large",
			modify:  func(_ *testing.T, p *fleet.EJBCA) { p.TrustCABundlePEM = strings.Repeat("a", ejbcaMaxPEMLength+1) },
			wantErr: "trust CA bundle is too large",
		},
		{name: "no PKCS#12", modify: func(_ *testing.T, p *fleet.EJBCA) { p.ClientP12 = "" }, wantErr: "client certificate PKCS#12 cannot be empty"},
		{name: "no PKCS#12 password", modify: func(_ *testing.T, p *fleet.EJBCA) { p.Password = "" }, wantErr: "PKCS#12 password cannot be empty"},
		{name: "PKCS#12 not base64", modify: func(_ *testing.T, p *fleet.EJBCA) { p.ClientP12 = "%%%" }, wantErr: "must be base64 encoded"},
		{
			name: "not a PKCS#12",
			modify: func(_ *testing.T, p *fleet.EJBCA) {
				p.ClientP12 = base64.StdEncoding.EncodeToString([]byte("not a p12"))
			},
			wantErr: "Couldn't read the PKCS#12 file",
		},
		{name: "wrong PKCS#12 password", modify: func(_ *testing.T, p *fleet.EJBCA) { p.Password = "wrong" }, wantErr: "PKCS#12 password is incorrect"},
		{
			name: "expired client certificate",
			modify: func(t *testing.T, p *fleet.EJBCA) {
				p.ClientP12 = ejbcaClientP12(t, time.Now().Add(-48*time.Hour), time.Now().Add(-time.Hour))
			},
			wantErr: "The client certificate expired on " + time.Now().Add(-time.Hour).UTC().Format(time.DateOnly),
		},
		{
			name: "client certificate not valid yet",
			modify: func(t *testing.T, p *fleet.EJBCA) {
				p.ClientP12 = ejbcaClientP12(t, time.Now().Add(48*time.Hour), time.Now().Add(72*time.Hour))
			},
			wantErr: "The client certificate isn't valid until " + time.Now().Add(48*time.Hour).UTC().Format(time.DateOnly),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := newEJBCATestEnv(t, nil)
			payload := validEJBCA(t)
			tt.modify(t, &payload)

			_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
			require.ErrorContains(t, err, tt.wantErr)
			require.NotErrorIs(t, err, errStopBeforeActivity)
			assert.Nil(t, env.created, "nothing is stored")
			assert.Empty(t, env.verified, "EJBCA is not contacted with an invalid request")
		})
	}

	t.Run("EJBCA rejects the connection", func(t *testing.T) {
		t.Parallel()
		env := newEJBCATestEnv(t, nil)
		env.ejbca.VerifyConnectionFunc = func(context.Context, fleet.EJBCA) error {
			return errors.New(`EJBCA CA "ManagementCA" was not found.`)
		}
		payload := validEJBCA(t)

		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
		require.EqualError(t, err, `Couldn't add certificate authority. EJBCA CA "ManagementCA" was not found.`)
		var badRequest *fleet.BadRequestError
		require.ErrorAs(t, err, &badRequest)
		assert.Nil(t, env.created, "nothing is stored")
	})

	t.Run("only one CA can be created at a time", func(t *testing.T) {
		t.Parallel()
		env := newEJBCATestEnv(t, nil)
		payload := validEJBCA(t)
		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload, DigiCert: &fleet.DigiCertCA{}})
		require.ErrorContains(t, err, "Only one certificate authority can be created at a time")
	})
}

func TestUpdateEJBCACertificateAuthority(t *testing.T) {
	t.Parallel()

	storedCertPEM, storedKeyPEM := "stored-cert", "stored-key"
	stored := func() *fleet.CertificateAuthority {
		return &fleet.CertificateAuthority{
			ID:                            7,
			Type:                          string(fleet.CATypeEJBCA),
			Name:                          new("EJBCA_WIFI"),
			URL:                           new("https://ejbca.example.com:8443"),
			MTLSCertPEM:                   &storedCertPEM,
			MTLSKeyPEM:                    &storedKeyPEM,
			TrustCABundlePEM:              new("stored-bundle"),
			EJBCACAName:                   new("ManagementCA"),
			EJBCACertificateProfileName:   new("fleetCertProfile"),
			EJBCAEndEntityProfileName:     new("fleetEEProfile"),
			CertificateCommonName:         new("$FLEET_VAR_HOST_HARDWARE_SERIAL"),
			CertificateUserPrincipalNames: &[]string{"upn@example.com"},
		}
	}

	tests := []struct {
		name   string
		update func(t *testing.T) fleet.EJBCAUpdatePayload
		// wantVerified is the connection check that is expected, with the stored credential unless the update replaces it.
		wantVerified   bool
		checkVerified  func(t *testing.T, config fleet.EJBCA)
		checkUpdated   func(t *testing.T, ca *fleet.CertificateAuthority)
		verifyErr      error
		wantErr        string
		wantNotUpdated bool
	}{
		{
			name: "templates only don't need the connection",
			update: func(*testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{
					CertificateCommonName:         new("$FLEET_VAR_HOST_END_USER_EMAIL_IDP"),
					UsernameTemplate:              new("$FLEET_VAR_HOST_UUID"),
					CertificateUserPrincipalNames: &[]string{"$FLEET_VAR_HOST_END_USER_EMAIL_IDP"},
					EndEntityProfileName:          new("otherEEProfile"),
				}
			},
			checkUpdated: func(t *testing.T, ca *fleet.CertificateAuthority) {
				assert.Equal(t, "$FLEET_VAR_HOST_END_USER_EMAIL_IDP", *ca.CertificateCommonName)
				assert.Equal(t, "$FLEET_VAR_HOST_UUID", *ca.EJBCAUsernameTemplate)
				assert.Equal(t, "otherEEProfile", *ca.EJBCAEndEntityProfileName)
				assert.Nil(t, ca.MTLSCertPEM)
				assert.Nil(t, ca.MTLSKeyPEM)
				assert.Nil(t, ca.URL)
			},
		},
		{
			name: "certificate profile change verifies with the stored credential",
			update: func(*testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{CertificateProfileName: new("newProfile")}
			},
			wantVerified: true,
			checkVerified: func(t *testing.T, config fleet.EJBCA) {
				assert.Equal(t, "newProfile", config.CertificateProfileName)
				assert.Equal(t, storedCertPEM, config.MTLSCertPEM)
				assert.Equal(t, storedKeyPEM, config.MTLSKeyPEM)
				assert.Equal(t, "stored-bundle", config.TrustCABundlePEM)
				assert.Equal(t, "ManagementCA", config.CertificateAuthorityName)
			},
			checkUpdated: func(t *testing.T, ca *fleet.CertificateAuthority) {
				assert.Equal(t, "newProfile", *ca.EJBCACertificateProfileName)
				assert.Nil(t, ca.MTLSCertPEM, "the stored credential is untouched")
				assert.Nil(t, ca.MTLSKeyPEM)
			},
		},
		{
			name: "masked credentials keep the stored ones",
			update: func(*testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{
					ClientP12: new(fleet.MaskedPassword), Password: new(fleet.MaskedPassword), URL: new("https://new.example.com"),
				}
			},
			wantVerified: true,
			checkVerified: func(t *testing.T, config fleet.EJBCA) {
				assert.Equal(t, "https://new.example.com", config.URL)
				assert.Equal(t, storedCertPEM, config.MTLSCertPEM)
			},
			checkUpdated: func(t *testing.T, ca *fleet.CertificateAuthority) {
				assert.Equal(t, "https://new.example.com", *ca.URL)
				assert.Nil(t, ca.MTLSCertPEM)
			},
		},
		{
			name: "rotation replaces the credential once it verifies",
			update: func(t *testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{ClientP12: new(validEJBCAClientP12(t)), Password: new(ejbcaP12Password)}
			},
			wantVerified: true,
			checkVerified: func(t *testing.T, config fleet.EJBCA) {
				assert.NotEqual(t, storedCertPEM, config.MTLSCertPEM)
				assert.Contains(t, config.MTLSKeyPEM, "PRIVATE KEY")
			},
			checkUpdated: func(t *testing.T, ca *fleet.CertificateAuthority) {
				require.NotNil(t, ca.MTLSCertPEM)
				require.NotNil(t, ca.MTLSKeyPEM)
				assert.Contains(t, *ca.MTLSCertPEM, "BEGIN CERTIFICATE")
				assert.Contains(t, *ca.MTLSKeyPEM, "PRIVATE KEY")
			},
		},
		{
			name: "a replacement credential EJBCA rejects is not stored",
			update: func(t *testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{ClientP12: new(validEJBCAClientP12(t)), Password: new(ejbcaP12Password)}
			},
			wantVerified:   true,
			verifyErr:      errors.New("EJBCA rejected the TLS client certificate"),
			wantErr:        "Couldn't edit certificate authority. EJBCA rejected the TLS client certificate",
			wantNotUpdated: true,
		},
		{
			name: "PKCS#12 without password",
			update: func(t *testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{ClientP12: new(validEJBCAClientP12(t))}
			},
			wantErr:        "The PKCS#12 password must be set with client_p12",
			wantNotUpdated: true,
		},
		{
			name:           "password without PKCS#12",
			update:         func(*testing.T) fleet.EJBCAUpdatePayload { return fleet.EJBCAUpdatePayload{Password: new("x")} },
			wantErr:        "The password can only be set with client_p12",
			wantNotUpdated: true,
		},
		{
			name: "wrong password for the new PKCS#12",
			update: func(t *testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{ClientP12: new(validEJBCAClientP12(t)), Password: new("wrong")}
			},
			wantErr:        "PKCS#12 password is incorrect",
			wantNotUpdated: true,
		},
		{
			name: "expired replacement credential",
			update: func(t *testing.T) fleet.EJBCAUpdatePayload {
				p12 := ejbcaClientP12(t, time.Now().Add(-48*time.Hour), time.Now().Add(-time.Hour))
				return fleet.EJBCAUpdatePayload{ClientP12: &p12, Password: new(ejbcaP12Password)}
			},
			wantErr:        "The client certificate expired on",
			wantNotUpdated: true,
		},
		{
			name: "required field cleared",
			update: func(*testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{CertificateProfileName: new(" ")}
			},
			wantErr:        "certificate profile name cannot be empty",
			wantNotUpdated: true,
		},
		{
			name: "second UPN",
			update: func(*testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{CertificateUserPrincipalNames: &[]string{"a@example.com", "b@example.com"}}
			},
			wantErr:        "only one item can be added to certificate_user_principal_names",
			wantNotUpdated: true,
		},
		{
			name:           "invalid name",
			update:         func(*testing.T) fleet.EJBCAUpdatePayload { return fleet.EJBCAUpdatePayload{Name: new("bad name")} },
			wantErr:        "Invalid characters in the \"name\" field",
			wantNotUpdated: true,
		},
		{
			name: "only masked credentials",
			update: func(*testing.T) fleet.EJBCAUpdatePayload {
				return fleet.EJBCAUpdatePayload{ClientP12: new(fleet.MaskedPassword), Password: new(fleet.MaskedPassword)}
			},
			wantErr:        "EJBCA CA update payload is empty",
			wantNotUpdated: true,
		},
		{
			name:           "empty update",
			update:         func(*testing.T) fleet.EJBCAUpdatePayload { return fleet.EJBCAUpdatePayload{} },
			wantErr:        "EJBCA CA update payload is empty",
			wantNotUpdated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env := newEJBCATestEnv(t, stored())
			env.ejbca.VerifyConnectionFunc = func(_ context.Context, config fleet.EJBCA) error {
				env.verified = append(env.verified, config)
				return tt.verifyErr
			}
			update := tt.update(t)

			err := env.svc.UpdateCertificateAuthority(env.ctx, 7, fleet.CertificateAuthorityUpdatePayload{EJBCAUpdatePayload: &update})
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.ErrorIs(t, err, errStopBeforeActivity)
			}
			if tt.wantNotUpdated {
				assert.Nil(t, env.updated, "nothing is stored")
			} else {
				require.NotNil(t, env.updated)
				assert.Equal(t, string(fleet.CATypeEJBCA), env.updated.Type)
				tt.checkUpdated(t, env.updated)
			}
			if !tt.wantVerified {
				assert.Empty(t, env.verified)
				return
			}
			require.Len(t, env.verified, 1)
			if tt.checkVerified != nil {
				tt.checkVerified(t, env.verified[0])
			}
		})
	}

	t.Run("a CA of another type", func(t *testing.T) {
		t.Parallel()
		digicert := &fleet.CertificateAuthority{ID: 7, Type: string(fleet.CATypeDigiCert), Name: new("DigiCert")}
		env := newEJBCATestEnv(t, digicert)
		err := env.svc.UpdateCertificateAuthority(env.ctx, 7, fleet.CertificateAuthorityUpdatePayload{
			EJBCAUpdatePayload: &fleet.EJBCAUpdatePayload{CertificateProfileName: new("x")},
		})
		require.ErrorContains(t, err, "The certificate authority types must be the same.")
		assert.Nil(t, env.updated)
	})
}
