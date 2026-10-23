package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	googlecas_mock "github.com/fleetdm/fleet/v4/server/mock/googlecas"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/require"
)

const testGoogleCASKey = `{"type":"service_account","client_email":"sa@my-project.iam.gserviceaccount.com","private_key":"-----BEGIN PRIVATE KEY-----"}`

func goodGoogleCAS() *fleet.GoogleCASCA {
	return &fleet.GoogleCASCA{Name: "GOOGLE_WIFI", Project: "my-project", Location: "us-central1", CAPool: "my-ca-pool", APIKeyJSON: testGoogleCASKey}
}

// newGoogleCASTestService builds a service whose datastore records the CA it is asked to create or update and then fails, which keeps
// the activity recording, which can't be mocked here, from running.
func newGoogleCASTestService(t *testing.T, stored *fleet.CertificateAuthority) (*Service, *mock.Store, *googlecas_mock.Service, context.Context) {
	ds := new(mock.Store)
	ds.NewCertificateAuthorityFunc = func(ctx context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
		return nil, errors.New("mock error to avoid NewActivity panic")
	}
	ds.GetCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
		if stored != nil && stored.ID == id {
			return stored, nil
		}
		return nil, common_mysql.NotFound("certificate authority")
	}
	ds.UpdateCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, ca *fleet.CertificateAuthority) error {
		return errors.New("mock error to avoid NewActivity panic")
	}
	googleCAS := &googlecas_mock.Service{VerifyCAPoolFunc: func(ctx context.Context, ca fleet.GoogleCASCA) error { return nil }}
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{
		logger:           slog.New(slog.NewTextHandler(os.Stdout, nil)),
		ds:               ds,
		authz:            authorizer,
		googleCASService: googleCAS,
	}
	svc.config.Server.PrivateKey = "supersecret"
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	return svc, ds, googleCAS, ctx
}

func TestCreateGoogleCASCertificateAuthority(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		svc, ds, googleCAS, ctx := newGoogleCASTestService(t, nil)
		var created *fleet.CertificateAuthority
		ds.NewCertificateAuthorityFunc = func(ctx context.Context, ca *fleet.CertificateAuthority) (*fleet.CertificateAuthority, error) {
			created = ca
			return nil, errors.New("mock error to avoid NewActivity panic")
		}
		var verified fleet.GoogleCASCA
		googleCAS.VerifyCAPoolFunc = func(ctx context.Context, ca fleet.GoogleCASCA) error {
			verified = ca
			return nil
		}

		in := goodGoogleCAS()
		in.Name, in.Project = " GOOGLE_WIFI ", " my-project " // trimmed by Preprocess
		_, err := svc.NewCertificateAuthority(ctx, fleet.CertificateAuthorityPayload{GoogleCAS: in})
		require.EqualError(t, err, "mock error to avoid NewActivity panic")

		require.Equal(t, *goodGoogleCAS(), verified)
		require.Equal(t, string(fleet.CATypeGoogleCAS), created.Type)
		require.Equal(t, "GOOGLE_WIFI", *created.Name)
		require.Empty(t, *created.URL) // the column is NOT NULL
		require.Equal(t, "my-project", *created.GoogleCASProject)
		require.Equal(t, "us-central1", *created.GoogleCASLocation)
		require.Equal(t, "my-ca-pool", *created.GoogleCASCAPool)
		require.JSONEq(t, testGoogleCASKey, *created.GoogleCASAPIKeyJSON)
	})

	t.Run("validation", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			mutate  func(ca *fleet.GoogleCASCA)
			wantErr string
		}{
			{"bad name", func(ca *fleet.GoogleCASCA) { ca.Name = "a/b" }, "Only letters, numbers and underscores allowed"},
			{"empty project", func(ca *fleet.GoogleCASCA) { ca.Project = " " }, "The Google Cloud project ID cannot be empty."},
			{"empty location", func(ca *fleet.GoogleCASCA) { ca.Location = "" }, "The location cannot be empty."},
			{"empty ca pool", func(ca *fleet.GoogleCASCA) { ca.CAPool = "" }, "The CA pool cannot be empty."},
			{"empty key", func(ca *fleet.GoogleCASCA) { ca.APIKeyJSON = "" }, "The service account key cannot be empty."},
			{"masked key", func(ca *fleet.GoogleCASCA) { ca.APIKeyJSON = fleet.MaskedPassword }, "The service account key must be provided in full, not masked."},
			{"key is not json", func(ca *fleet.GoogleCASCA) { ca.APIKeyJSON = "nope" }, "The service account key must be a JSON object."},
			{"key too large", func(ca *fleet.GoogleCASCA) { ca.APIKeyJSON = fleet.GoogleCASKeyJSON(strings.Repeat("a", 16<<10+1)) }, "The service account key is too large."},
			{"pool with a slash", func(ca *fleet.GoogleCASCA) { ca.CAPool = "pool/certificates/other" }, "The CA pool can only contain letters, numbers, hyphens and underscores."},
			{"location with a slash", func(ca *fleet.GoogleCASCA) { ca.Location = "us-central1/../x" }, "The location can only contain lowercase letters, numbers and hyphens."},
			{"project with a query", func(ca *fleet.GoogleCASCA) { ca.Project = "p?x=1" }, "The Google Cloud project ID can only contain"},
			{"key is not a service account", func(ca *fleet.GoogleCASCA) { ca.APIKeyJSON = `{"type":"authorized_user"}` }, `"type" set to "service_account"`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				svc, _, googleCAS, ctx := newGoogleCASTestService(t, nil)
				in := goodGoogleCAS()
				tc.mutate(in)
				_, err := svc.NewCertificateAuthority(ctx, fleet.CertificateAuthorityPayload{GoogleCAS: in})
				require.ErrorContains(t, err, "Couldn't add certificate authority. ")
				require.ErrorContains(t, err, tc.wantErr)
				require.False(t, googleCAS.VerifyCAPoolFuncInvoked, "Google must not be contacted for a CA that fails validation")
			})
		}
	})

	t.Run("CA pool Google rejects is not stored", func(t *testing.T) {
		svc, ds, googleCAS, ctx := newGoogleCASTestService(t, nil)
		googleCAS.VerifyCAPoolFunc = func(ctx context.Context, ca fleet.GoogleCASCA) error {
			return errors.New(`CA pool "my-ca-pool" wasn't found in project "my-project", location "us-central1".`)
		}
		_, err := svc.NewCertificateAuthority(ctx, fleet.CertificateAuthorityPayload{GoogleCAS: goodGoogleCAS()})
		require.EqualError(t, err, `Couldn't add certificate authority. CA pool "my-ca-pool" wasn't found in project "my-project", location "us-central1".`)
		require.False(t, ds.NewCertificateAuthorityFuncInvoked)
	})
}

func TestUpdateGoogleCASCertificateAuthority(t *testing.T) {
	t.Parallel()

	stored := func() *fleet.CertificateAuthority {
		return &fleet.CertificateAuthority{
			ID:                  1,
			Type:                string(fleet.CATypeGoogleCAS),
			Name:                new("GOOGLE_WIFI"),
			URL:                 new(""),
			GoogleCASProject:    new("my-project"),
			GoogleCASLocation:   new("us-central1"),
			GoogleCASCAPool:     new("my-ca-pool"),
			GoogleCASAPIKeyJSON: new(testGoogleCASKey),
		}
	}
	update := func(p fleet.GoogleCASCAUpdatePayload) fleet.CertificateAuthorityUpdatePayload {
		return fleet.CertificateAuthorityUpdatePayload{GoogleCASCAUpdatePayload: &p}
	}

	t.Run("name change is stored without asking Google, even with the other values sent unchanged", func(t *testing.T) {
		svc, ds, googleCAS, ctx := newGoogleCASTestService(t, stored())
		var updated *fleet.CertificateAuthority
		ds.UpdateCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, ca *fleet.CertificateAuthority) error {
			updated = ca
			return errors.New("mock error to avoid NewActivity panic")
		}
		err := svc.UpdateCertificateAuthority(ctx, 1, update(fleet.GoogleCASCAUpdatePayload{
			Name: new("GOOGLE_VPN"), Project: new("my-project"), Location: new("us-central1"), CAPool: new("my-ca-pool"),
			APIKeyJSON: new(fleet.GoogleCASKeyJSON(fleet.MaskedPassword)),
		}))
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		require.False(t, googleCAS.VerifyCAPoolFuncInvoked)
		require.Equal(t, &fleet.CertificateAuthority{Type: string(fleet.CATypeGoogleCAS), Name: new("GOOGLE_VPN")}, updated)
	})

	t.Run("pool change is verified with the stored key and only the change is stored", func(t *testing.T) {
		svc, ds, googleCAS, ctx := newGoogleCASTestService(t, stored())
		var verified fleet.GoogleCASCA
		googleCAS.VerifyCAPoolFunc = func(ctx context.Context, ca fleet.GoogleCASCA) error {
			verified = ca
			return nil
		}
		var updated *fleet.CertificateAuthority
		ds.UpdateCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, ca *fleet.CertificateAuthority) error {
			updated = ca
			return errors.New("mock error to avoid NewActivity panic")
		}
		err := svc.UpdateCertificateAuthority(ctx, 1, update(fleet.GoogleCASCAUpdatePayload{CAPool: new("other-pool"), APIKeyJSON: new(fleet.GoogleCASKeyJSON(fleet.MaskedPassword))}))
		require.EqualError(t, err, "mock error to avoid NewActivity panic")
		want := *goodGoogleCAS()
		want.CAPool = "other-pool"
		require.Equal(t, want, verified)
		require.Equal(t, &fleet.CertificateAuthority{Type: string(fleet.CATypeGoogleCAS), GoogleCASCAPool: new("other-pool")}, updated)
	})

	t.Run("rejected by Google is not stored", func(t *testing.T) {
		svc, ds, googleCAS, ctx := newGoogleCASTestService(t, stored())
		googleCAS.VerifyCAPoolFunc = func(ctx context.Context, ca fleet.GoogleCASCA) error { return errors.New("no access") }
		rotated := strings.Replace(testGoogleCASKey, `"sa@`, `"rotated@`, 1)
		err := svc.UpdateCertificateAuthority(ctx, 1, update(fleet.GoogleCASCAUpdatePayload{APIKeyJSON: new(fleet.GoogleCASKeyJSON(rotated))}))
		require.EqualError(t, err, "Couldn't edit certificate authority. no access")
		require.False(t, ds.UpdateCertificateAuthorityByIDFuncInvoked)
	})

	t.Run("nothing changed is neither verified nor stored", func(t *testing.T) {
		var reformattedKey bytes.Buffer
		require.NoError(t, json.Indent(&reformattedKey, []byte(testGoogleCASKey), "", "  "))
		for _, tc := range []struct {
			name    string
			payload fleet.GoogleCASCAUpdatePayload
		}{
			{"empty payload", fleet.GoogleCASCAUpdatePayload{}},
			{"masked key alone", fleet.GoogleCASCAUpdatePayload{APIKeyJSON: new(fleet.GoogleCASKeyJSON(fleet.MaskedPassword))}},
			{"stored values sent again", fleet.GoogleCASCAUpdatePayload{
				Name: new("GOOGLE_WIFI"), Project: new(" my-project "), Location: new("us-central1"), CAPool: new("my-ca-pool"),
				APIKeyJSON: new(fleet.GoogleCASKeyJSON(reformattedKey.String())),
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				svc, ds, googleCAS, ctx := newGoogleCASTestService(t, stored())
				require.NoError(t, svc.UpdateCertificateAuthority(ctx, 1, update(tc.payload)))
				require.False(t, googleCAS.VerifyCAPoolFuncInvoked)
				require.False(t, ds.UpdateCertificateAuthorityByIDFuncInvoked)
			})
		}
	})

	t.Run("validation", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			payload fleet.GoogleCASCAUpdatePayload
			wantErr string
		}{
			{"bad name", fleet.GoogleCASCAUpdatePayload{Name: new("NDES")}, "CA name cannot be NDES"},
			{"empty project", fleet.GoogleCASCAUpdatePayload{Project: new("")}, "The Google Cloud project ID cannot be empty."},
			{"bad key", fleet.GoogleCASCAUpdatePayload{APIKeyJSON: new(fleet.GoogleCASKeyJSON("{}"))}, `"type" set to "service_account"`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				svc, ds, _, ctx := newGoogleCASTestService(t, stored())
				err := svc.UpdateCertificateAuthority(ctx, 1, update(tc.payload))
				require.ErrorContains(t, err, "Couldn't edit certificate authority. ")
				require.ErrorContains(t, err, tc.wantErr)
				require.False(t, ds.UpdateCertificateAuthorityByIDFuncInvoked)
			})
		}
	})

	t.Run("type cannot change", func(t *testing.T) {
		hydrant := stored()
		hydrant.Type = string(fleet.CATypeHydrant)
		svc, _, _, ctx := newGoogleCASTestService(t, hydrant)
		err := svc.UpdateCertificateAuthority(ctx, 1, update(fleet.GoogleCASCAUpdatePayload{Name: new("GOOGLE_VPN")}))
		require.EqualError(t, err, "Couldn't edit certificate authority. The certificate authority types must be the same.")
	})
}

func TestProcessGoogleCASCAs(t *testing.T) {
	t.Parallel()
	svc, _, googleCAS, ctx := newGoogleCASTestService(t, nil)

	existing := []fleet.GoogleCASCA{*goodGoogleCAS(), {Name: "GONE", Project: "p", Location: "l", CAPool: "c", APIKeyJSON: testGoogleCASKey}}
	changed := *goodGoogleCAS()
	changed.CAPool = "renewed-pool"
	added := fleet.GoogleCASCA{Name: "NEW", Project: "p", Location: "l", CAPool: "c", APIKeyJSON: testGoogleCASKey}

	var verified []string
	googleCAS.VerifyCAPoolFunc = func(ctx context.Context, ca fleet.GoogleCASCA) error {
		verified = append(verified, ca.Name)
		return nil
	}
	// An unchanged CA is neither verified nor updated, so GitOps records no edited activity for it.
	unchanged := existing[0]
	unchanged.ID = 0 // incoming CAs from YAML have no ID
	ops := &fleet.CertificateAuthoritiesBatchOperations{}
	require.NoError(t, svc.processGoogleCASCAs(ctx, ops, []fleet.GoogleCASCA{unchanged, added}, existing))
	require.Equal(t, []string{"NEW"}, verified)
	require.Empty(t, ops.Update)

	verified = nil
	ops = &fleet.CertificateAuthoritiesBatchOperations{}
	require.NoError(t, svc.processGoogleCASCAs(ctx, ops, []fleet.GoogleCASCA{changed, added}, existing))
	require.ElementsMatch(t, []string{"GOOGLE_WIFI", "NEW"}, verified)

	names := func(cas []*fleet.CertificateAuthority) []string {
		var out []string
		for _, ca := range cas {
			require.Equal(t, string(fleet.CATypeGoogleCAS), ca.Type)
			require.Empty(t, *ca.URL)
			out = append(out, *ca.Name)
		}
		return out
	}
	require.Equal(t, []string{"NEW"}, names(ops.Add))
	require.Equal(t, []string{"GOOGLE_WIFI"}, names(ops.Update))
	require.Equal(t, "renewed-pool", *ops.Update[0].GoogleCASCAPool)
	require.Equal(t, []string{"GONE"}, names(ops.Delete))

	// A CA that fails validation stops the batch, and a masked key is refused in YAML since the full key is the intended input.
	for key, wantErr := range map[string]string{
		"nope":               "certificate_authorities.google_cas: The service account key must be a JSON object.",
		fleet.MaskedPassword: "certificate_authorities.google_cas: The service account key must be provided in full, not masked.",
	} {
		bad := added
		bad.APIKeyJSON = fleet.GoogleCASKeyJSON(key)
		err := svc.processGoogleCASCAs(ctx, &fleet.CertificateAuthoritiesBatchOperations{}, []fleet.GoogleCASCA{bad}, nil)
		require.ErrorContains(t, err, wantErr)
	}
}
