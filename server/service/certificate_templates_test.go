package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCreateCertificateTemplate(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	type TestCAID uint

	const (
		InvalidCATypeID TestCAID = iota + 1
		ValidCATypeID
	)

	const TeamID = 1

	ds.GetCertificateAuthorityByIDFunc = func(ctx context.Context, id uint, includeSecrets bool) (*fleet.CertificateAuthority, error) {
		if id == uint(InvalidCATypeID) {
			ca := fleet.CertificateAuthority{
				ID:   id,
				Type: string(fleet.CATypeDigiCert),
			}
			return &ca, nil
		}
		if id == uint(ValidCATypeID) {
			ca := fleet.CertificateAuthority{
				ID:   id,
				Type: string(fleet.CATypeCustomSCEPProxy),
			}
			return &ca, nil
		}
		return nil, errors.New("not found")
	}

	ds.CreateCertificateTemplateFunc = func(ctx context.Context, certificateTemplate *fleet.CertificateTemplate) (*fleet.CertificateTemplateResponseFull, error) {
		return nil, nil
	}
	t.Run("Invalid CA type", func(t *testing.T) {
		_, err := svc.CreateCertificateTemplate(ctx, "my template", TeamID, uint(InvalidCATypeID), "CN=$FLEET_VAR_HOST_UUID")
		require.Error(t, err)
		// Check that the error is about invalid CA type
		require.Contains(t, err.Error(), "Currently, only the custom_scep_proxy certificate authority is supported")
	})

	t.Run("Valid CA type", func(t *testing.T) {
		_, err := svc.CreateCertificateTemplate(ctx, "my template", TeamID, uint(ValidCATypeID), "CN=$FLEET_VAR_HOST_UUID")
		require.NoError(t, err)
	})

	t.Run("Missing CA", func(t *testing.T) {
		_, err := svc.CreateCertificateTemplate(ctx, "my template", TeamID, 999, "CN=$FLEET_VAR_HOST_UUID")
		require.Error(t, err)
		// Check that the error is about invalid CA type
		require.Contains(t, err.Error(), "not found")
	})
}

func TestApplyCertificateTemplateSpecs(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.TeamLiteFunc = func(ctx context.Context, id uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{
			ID:   id,
			Name: "Test Team",
		}, nil
	}

	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time) error {
		return nil
	}

	// Set up certificate authority mocks
	certAuthorities := []*fleet.CertificateAuthority{
		{
			ID:        1,
			Name:      ptr.String("Test CA 1"),
			Type:      string(fleet.CATypeCustomSCEPProxy),
			URL:       ptr.String("https://ca1.example.com"),
			Challenge: ptr.String("challenge1"),
		},
		{
			ID:        2,
			Name:      ptr.String("Test CA 2"),
			Type:      string(fleet.CATypeCustomSCEPProxy),
			URL:       ptr.String("https://ca2.example.com"),
			Challenge: ptr.String("challenge2"),
		},
		{
			ID:                            3,
			Name:                          ptr.String("Test CA 3"),
			Type:                          string(fleet.CATypeDigiCert),
			URL:                           ptr.String("https://ca3.example.com"),
			Challenge:                     ptr.String("challenge3"),
			CertificateCommonName:         ptr.String("foo"),
			CertificateSeatID:             ptr.String("foo"),
			CertificateUserPrincipalNames: &[]string{"foo"},
			APIToken:                      ptr.String("foo"),
			ProfileID:                     ptr.String("foo"),
		},
	}

	ds.ListCertificateAuthoritiesFunc = func(ctx context.Context) ([]*fleet.CertificateAuthoritySummary, error) {
		summaries := make([]*fleet.CertificateAuthoritySummary, 0, len(certAuthorities))
		for _, ca := range certAuthorities {
			summaries = append(summaries, &fleet.CertificateAuthoritySummary{
				ID:   ca.ID,
				Name: *ca.Name,
				Type: ca.Type,
			})
		}
		return summaries, nil
	}

	// Track certificate templates that are created
	var createdCertificates []fleet.CertificateTemplate

	ds.BatchUpsertCertificateTemplatesFunc = func(ctx context.Context, certificates []*fleet.CertificateTemplate) (map[uint]bool, error) {
		createdCertificates = nil
		createdMap := make(map[uint]bool)
		for _, cert := range certificates {
			createdCertificates = append(createdCertificates, *cert)
			createdMap[cert.TeamID] = true
		}
		return createdMap, nil
	}

	t.Run("Valid CA types", func(t *testing.T) {
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   "Template 1",
				CertificateAuthorityId: 1,
				SubjectName:            "foo",
			},
			{
				Name:                   "Template 2",
				CertificateAuthorityId: 2,
				SubjectName:            "bar",
			},
		})
		require.NoError(t, err)
		require.Len(t, createdCertificates, 2)
		require.Equal(t, "Template 1", createdCertificates[0].Name)
		require.Equal(t, uint(1), createdCertificates[0].CertificateAuthorityID)
		require.Equal(t, "Template 2", createdCertificates[1].Name)
		require.Equal(t, uint(2), createdCertificates[1].CertificateAuthorityID)
	})

	t.Run("Invalid CA type", func(t *testing.T) {
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   "Template 3",
				CertificateAuthorityId: 3,
				SubjectName:            "baz",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Currently, only the custom_scep_proxy certificate authority is supported")
	})

	t.Run("Missing CA", func(t *testing.T) {
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   "Template 4",
				CertificateAuthorityId: 4,
				SubjectName:            "baz",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}
