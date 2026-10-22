package service

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/ejbca/ejbcatest"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	ejbca_mock "github.com/fleetdm/fleet/v4/server/mock/ejbca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errStopBeforeActivity makes the datastore mock end a create or update before it records an activity, which the unit tests can't.
var errStopBeforeActivity = errors.New("mock error to avoid NewActivity panic")

const ejbcaP12Password = "p12-password"

// ejbcaP12s are client certificates in base64 PKCS#12 files, built once because encrypting them is slow.
type ejbcaP12s struct {
	valid, expired, notYetValid string
	expiredAt, validFrom        time.Time
}

func newEJBCAP12s(t *testing.T) ejbcaP12s {
	t.Helper()
	ca := ejbcatest.NewCA(t, "CA")
	now := time.Now()
	f := ejbcaP12s{expiredAt: now.Add(-time.Hour), validFrom: now.Add(48 * time.Hour)}
	f.valid = ejbcatest.ClientP12Between(t, ca, now.Add(-time.Hour), now.Add(24*time.Hour), ejbcaP12Password)
	f.expired = ejbcatest.ClientP12Between(t, ca, now.Add(-48*time.Hour), f.expiredAt, ejbcaP12Password)
	f.notYetValid = ejbcatest.ClientP12Between(t, ca, f.validFrom, now.Add(72*time.Hour), ejbcaP12Password)
	return f
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
	ds := new(mock.Store)
	env := &ejbcaTestEnv{ejbca: new(ejbca_mock.Service), ds: ds}
	env.ejbca.VerifyConnectionFunc = func(_ context.Context, config fleet.EJBCA) error {
		env.verified = append(env.verified, config)
		return nil
	}
	ds.NewCertificateAuthorityFunc = func(_ context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
		env.created = ca
		return nil, errStopBeforeActivity
	}
	ds.GetCertificateAuthorityByIDFunc = func(_ context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
		require.True(t, includeSecrets, "an update merges with the stored credential")
		return stored, nil
	}
	ds.UpdateCertificateAuthorityByIDFunc = func(_ context.Context, _ uint, ca *fleet.CertificateAuthority) error {
		env.updated = ca
		return errStopBeforeActivity
	}

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	env.svc = &Service{
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		ds:           ds,
		authz:        authorizer,
		ejbcaService: env.ejbca,
	}
	env.svc.config.Server.PrivateKey = "supersecret"
	env.ctx = viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	return env
}

func validEJBCA(p12 string) fleet.EJBCA {
	return fleet.EJBCA{
		Name:                          "EJBCA_WIFI",
		URL:                           "https://ejbca.example.com:8443",
		ClientP12:                     p12,
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

func TestValidateEJBCAFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		modify  func(p *fleet.EJBCA)
		wantErr string
	}{
		{name: "valid, with variables"},
		{name: "no CA name", modify: func(p *fleet.EJBCA) { p.CertificateAuthorityName = "" }, wantErr: "EJBCA CA name cannot be empty"},
		{
			name:    "no certificate profile name",
			modify:  func(p *fleet.EJBCA) { p.CertificateProfileName = " " },
			wantErr: "certificate profile name cannot be empty",
		},
		{
			name:    "no end entity profile name",
			modify:  func(p *fleet.EJBCA) { p.EndEntityProfileName = "" },
			wantErr: "end entity profile name cannot be empty",
		},
		{name: "no common name", modify: func(p *fleet.EJBCA) { p.CertificateCommonName = "" }, wantErr: "common name cannot be empty"},
		{
			name:    "username template with only spaces",
			modify:  func(p *fleet.EJBCA) { p.UsernameTemplate = "  " },
			wantErr: "username template cannot be only spaces",
		},
		{
			name:    "name too long",
			modify:  func(p *fleet.EJBCA) { p.CertificateProfileName = strings.Repeat("a", 256) },
			wantErr: "certificate profile name cannot be longer than 255 characters",
		},
		{
			name:    "unsupported variable in the common name",
			modify:  func(p *fleet.EJBCA) { p.CertificateCommonName = "$FLEET_VAR_HOST_END_USER_IDP_USERNAME_X" },
			wantErr: "FLEET_VAR_HOST_END_USER_IDP_USERNAME_X is not allowed in CA Common Name (CN)",
		},
		{
			name:    "unsupported variable in the username template",
			modify:  func(p *fleet.EJBCA) { p.UsernameTemplate = "$FLEET_VAR_DIGICERT_DATA_X" },
			wantErr: "FLEET_VAR_DIGICERT_DATA_X is not allowed in EJBCA Username Template",
		},
		{
			name:    "more than one UPN",
			modify:  func(p *fleet.EJBCA) { p.CertificateUserPrincipalNames = []string{"a@example.com", "b@example.com"} },
			wantErr: "only one item can be added to certificate_user_principal_names",
		},
		{
			name:    "blank UPN",
			modify:  func(p *fleet.EJBCA) { p.CertificateUserPrincipalNames = []string{" "} },
			wantErr: "EJBCA certificate_user_principal_name cannot be empty if specified",
		},
		{
			name:    "UPN too long",
			modify:  func(p *fleet.EJBCA) { p.CertificateUserPrincipalNames = []string{strings.Repeat("a", 256)} },
			wantErr: "user principal name cannot be longer than 255 characters",
		},
		{
			name:    "unsupported variable in the UPN",
			modify:  func(p *fleet.EJBCA) { p.CertificateUserPrincipalNames = []string{"$FLEET_VAR_NDES_SCEP_CHALLENGE"} },
			wantErr: "FLEET_VAR_NDES_SCEP_CHALLENGE is not allowed in CA User Principal Name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validEJBCA("")
			if tt.modify != nil {
				tt.modify(&p)
			}
			err := validateEJBCAFields(&p, "")
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestReadEJBCAClientP12(t *testing.T) {
	t.Parallel()
	p12s := newEJBCAP12s(t)
	tests := []struct {
		name     string
		p12      string
		password string
		wantErr  string
		// wantField is the field the error is on, client_p12 unless set.
		wantField string
	}{
		{name: "valid", p12: p12s.valid, password: ejbcaP12Password},
		{name: "no PKCS#12", password: ejbcaP12Password, wantErr: "client certificate PKCS#12 cannot be empty"},
		{name: "no password", p12: p12s.valid, wantErr: "PKCS#12 password cannot be empty", wantField: "password"},
		{name: "not base64", p12: "%%%", password: ejbcaP12Password, wantErr: "must be base64 encoded"},
		{
			name:     "not a PKCS#12",
			p12:      base64.StdEncoding.EncodeToString([]byte("not a p12")),
			password: ejbcaP12Password,
			wantErr:  "Couldn't read the PKCS#12 file",
		},
		{name: "wrong password", p12: p12s.valid, password: "wrong", wantErr: "PKCS#12 password is incorrect", wantField: "password"},
		{
			name:     "expired",
			p12:      p12s.expired,
			password: ejbcaP12Password,
			wantErr:  "The client certificate expired on " + p12s.expiredAt.UTC().Format(time.DateOnly),
		},
		{
			name:     "not valid yet",
			p12:      p12s.notYetValid,
			password: ejbcaP12Password,
			wantErr:  "The client certificate isn't valid until " + p12s.validFrom.UTC().Format(time.DateOnly),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := readEJBCAClientP12(t.Context(), tt.p12, tt.password, "")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				invalid, ok := errors.AsType[*fleet.InvalidArgumentError](err)
				require.True(t, ok, "got %v", err)
				assert.Equal(t, cmp.Or(tt.wantField, "client_p12"), invalid.Invalid()[0]["name"])
				return
			}
			require.NoError(t, err)
			assert.Contains(t, creds.KeyPEM, "PRIVATE KEY")
		})
	}
}

func TestNewEJBCACertificateAuthority(t *testing.T) {
	t.Parallel()
	p12s := newEJBCAP12s(t)

	t.Run("creates the CA from the PKCS#12 and verifies the connection", func(t *testing.T) {
		env := newEJBCATestEnv(t, nil)
		payload := validEJBCA(p12s.valid)

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

		// The certificate and the key come out of the PKCS#12.
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
		env := newEJBCATestEnv(t, nil)
		payload := validEJBCA(p12s.valid)
		payload.TrustCABundlePEM, payload.UsernameTemplate, payload.CertificateUserPrincipalNames = "", "", nil

		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
		require.ErrorIs(t, err, errStopBeforeActivity)
		assert.Nil(t, env.created.TrustCABundlePEM)
		assert.Nil(t, env.created.EJBCAUsernameTemplate)
		assert.Nil(t, env.created.CertificateUserPrincipalNames)
	})

	// The validators have their own tests, so these only show that a failure stops the create before EJBCA is contacted.
	tests := []struct {
		name    string
		modify  func(p *fleet.EJBCA)
		wantErr string
	}{
		{name: "invalid name", modify: func(p *fleet.EJBCA) { p.Name = "bad name" }, wantErr: "Invalid characters in the \"name\" field"},
		{name: "invalid URL", modify: func(p *fleet.EJBCA) { p.URL = "not a url" }, wantErr: "Invalid EJBCA URL"},
		{name: "invalid fields", modify: func(p *fleet.EJBCA) { p.CertificateAuthorityName = "" }, wantErr: "EJBCA CA name cannot be empty"},
		{
			name:    "trust bundle too large",
			modify:  func(p *fleet.EJBCA) { p.TrustCABundlePEM = strings.Repeat("a", ejbcaMaxPEMLength+1) },
			wantErr: "trust CA bundle is too large",
		},
		{name: "invalid PKCS#12", modify: func(p *fleet.EJBCA) { p.Password = "wrong" }, wantErr: "PKCS#12 password is incorrect"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEJBCATestEnv(t, nil)
			payload := validEJBCA(p12s.valid)
			tt.modify(&payload)

			_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
			require.ErrorContains(t, err, tt.wantErr)
			require.NotErrorIs(t, err, errStopBeforeActivity)
			assert.Nil(t, env.created, "nothing is stored")
			assert.Empty(t, env.verified, "EJBCA is not contacted with an invalid request")
		})
	}

	t.Run("EJBCA rejects the connection", func(t *testing.T) {
		env := newEJBCATestEnv(t, nil)
		env.ejbca.VerifyConnectionFunc = func(context.Context, fleet.EJBCA) error {
			return errors.New(`EJBCA CA "ManagementCA" was not found.`)
		}
		payload := validEJBCA(p12s.valid)

		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
		require.EqualError(t, err, `Couldn't add certificate authority. EJBCA CA "ManagementCA" was not found.`)
		var badRequest *fleet.BadRequestError
		require.ErrorAs(t, err, &badRequest)
		assert.Nil(t, env.created, "nothing is stored")
	})

	t.Run("the name is used by another CA", func(t *testing.T) {
		env := newEJBCATestEnv(t, nil)
		env.ds.NewCertificateAuthorityFunc = func(context.Context, *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
			return nil, fleet.EJBCANameInUseError{Name: "EJBCA_WIFI"}
		}
		payload := validEJBCA(p12s.valid)
		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload})
		var badRequest *fleet.BadRequestError
		require.ErrorAs(t, err, &badRequest)
		require.ErrorContains(t, err, `Couldn't add certificate authority. "EJBCA_WIFI" name is already used by another certificate authority`)
	})

	t.Run("only one CA can be created at a time", func(t *testing.T) {
		env := newEJBCATestEnv(t, nil)
		payload := validEJBCA(p12s.valid)
		_, err := env.svc.NewCertificateAuthority(env.ctx, fleet.CertificateAuthorityPayload{EJBCA: &payload, DigiCert: &fleet.DigiCertCA{}})
		require.ErrorContains(t, err, "Only one certificate authority can be created at a time")
	})
}

func TestUpdateEJBCACertificateAuthority(t *testing.T) {
	t.Parallel()
	p12s := newEJBCAP12s(t)

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
		update fleet.EJBCAUpdatePayload
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
			update: fleet.EJBCAUpdatePayload{
				CertificateCommonName:         new("$FLEET_VAR_HOST_END_USER_EMAIL_IDP"),
				UsernameTemplate:              new("$FLEET_VAR_HOST_UUID"),
				CertificateUserPrincipalNames: &[]string{"$FLEET_VAR_HOST_END_USER_EMAIL_IDP"},
				EndEntityProfileName:          new("otherEEProfile"),
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
			name:         "certificate profile change verifies with the stored credential",
			update:       fleet.EJBCAUpdatePayload{CertificateProfileName: new("newProfile")},
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
			update: fleet.EJBCAUpdatePayload{
				ClientP12: new(fleet.MaskedPassword), Password: new(fleet.MaskedPassword), URL: new("https://new.example.com"),
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
			name:         "rotation replaces the credential once it verifies",
			update:       fleet.EJBCAUpdatePayload{ClientP12: &p12s.valid, Password: new(ejbcaP12Password)},
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
			name:           "a replacement credential EJBCA rejects is not stored",
			update:         fleet.EJBCAUpdatePayload{ClientP12: &p12s.valid, Password: new(ejbcaP12Password)},
			wantVerified:   true,
			verifyErr:      errors.New("EJBCA rejected the TLS client certificate"),
			wantErr:        "Couldn't edit certificate authority. EJBCA rejected the TLS client certificate",
			wantNotUpdated: true,
		},
		{
			name:           "expired replacement credential",
			update:         fleet.EJBCAUpdatePayload{ClientP12: &p12s.expired, Password: new(ejbcaP12Password)},
			wantErr:        "The client certificate expired on",
			wantNotUpdated: true,
		},
		{
			name:           "PKCS#12 without password",
			update:         fleet.EJBCAUpdatePayload{ClientP12: &p12s.valid},
			wantErr:        "The PKCS#12 password must be set with client_p12",
			wantNotUpdated: true,
		},
		{
			name:           "password without PKCS#12",
			update:         fleet.EJBCAUpdatePayload{Password: new("x")},
			wantErr:        "The password can only be set with client_p12",
			wantNotUpdated: true,
		},
		{
			name:           "required field cleared",
			update:         fleet.EJBCAUpdatePayload{CertificateProfileName: new(" ")},
			wantErr:        "certificate profile name cannot be empty",
			wantNotUpdated: true,
		},
		{
			name:           "invalid name",
			update:         fleet.EJBCAUpdatePayload{Name: new("bad name")},
			wantErr:        "Invalid characters in the \"name\" field",
			wantNotUpdated: true,
		},
		{
			name:           "only masked credentials",
			update:         fleet.EJBCAUpdatePayload{ClientP12: new(fleet.MaskedPassword), Password: new(fleet.MaskedPassword)},
			wantErr:        "EJBCA CA update payload is empty",
			wantNotUpdated: true,
		},
		{
			name:           "empty update",
			update:         fleet.EJBCAUpdatePayload{},
			wantErr:        "EJBCA CA update payload is empty",
			wantNotUpdated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEJBCATestEnv(t, stored())
			env.ejbca.VerifyConnectionFunc = func(_ context.Context, config fleet.EJBCA) error {
				env.verified = append(env.verified, config)
				return tt.verifyErr
			}

			err := env.svc.UpdateCertificateAuthority(env.ctx, 7, fleet.CertificateAuthorityUpdatePayload{EJBCAUpdatePayload: &tt.update})
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
		digicert := &fleet.CertificateAuthority{ID: 7, Type: string(fleet.CATypeDigiCert), Name: new("DigiCert")}
		env := newEJBCATestEnv(t, digicert)
		err := env.svc.UpdateCertificateAuthority(env.ctx, 7, fleet.CertificateAuthorityUpdatePayload{
			EJBCAUpdatePayload: &fleet.EJBCAUpdatePayload{CertificateProfileName: new("x")},
		})
		require.ErrorContains(t, err, "The certificate authority types must be the same.")
		assert.Nil(t, env.updated)
	})
}
