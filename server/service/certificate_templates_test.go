package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
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

	ds.CreateCertificateTemplateFunc = func(ctx context.Context, certificateTemplate *fleet.CertificateTemplate) (*fleet.CertificateTemplateResponse, error) {
		return &fleet.CertificateTemplateResponse{
			CertificateTemplateResponseSummary: fleet.CertificateTemplateResponseSummary{
				ID:   1,
				Name: certificateTemplate.Name,
			},
		}, nil
	}
	ds.CreatePendingCertificateTemplatesForExistingHostsFunc = func(ctx context.Context, certificateTemplateID uint, teamID uint) (int64, error) {
		return 0, nil
	}
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{ID: tid, Name: "Yellow jackets"}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
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

	t.Run("Empty or whitespace-only name", func(t *testing.T) {
		whitespaceNames := []string{"", " ", "  ", "\t", "\n", "   \t\n  "}
		for _, name := range whitespaceNames {
			_, err := svc.CreateCertificateTemplate(ctx, name, TeamID, uint(ValidCATypeID), "CN=$FLEET_VAR_HOST_UUID")
			require.Error(t, err)
			require.Contains(t, err.Error(), "Certificate template name is required")
		}
	})

	t.Run("Name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 256)
		_, err := svc.CreateCertificateTemplate(ctx, longName, TeamID, uint(ValidCATypeID), "CN=$FLEET_VAR_HOST_UUID")
		require.Error(t, err)
		require.Contains(t, err.Error(), "Certificate template name is too long")
	})

	t.Run("Name with invalid characters", func(t *testing.T) {
		testCases := []struct {
			name string
		}{
			{name: "template@name"},
			{name: "template#name"},
			{name: "template$name"},
			{name: "template%name"},
			{name: "template.name"},
			{name: "template/name"},
			{name: "template\\name"},
			{name: "template!name"},
			{name: "template?name"},
			{name: "template*name"},
			{name: "template+name"},
			{name: "template=name"},
			{name: "template<name>"},
			{name: "template(name)"},
			{name: "template[name]"},
			{name: "template{name}"},
			{name: "template|name"},
			{name: "template;name"},
			{name: "template:name"},
			{name: "template'name"},
			{name: "template\"name"},
			{name: "template`name"},
			{name: "template~name"},
			{name: "template^name"},
			{name: "template	name"},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.CreateCertificateTemplate(ctx, tc.name, TeamID, uint(ValidCATypeID), "CN=$FLEET_VAR_HOST_UUID")
				require.Error(t, err)
				require.Contains(t, err.Error(), "Invalid certificate template name")
			})
		}
	})

	t.Run("Name with valid characters", func(t *testing.T) {
		validNames := []string{
			"my template",
			" my template ",
			"my-template",
			"my_template",
			"MyTemplate123",
			"Template 1",
			"UPPERCASE",
			"lowercase",
			"Mix-Ed_Case 123",
			"a",
			"1",
			"a1",
			"1a",
		}
		for _, name := range validNames {
			t.Run(name, func(t *testing.T) {
				_, err := svc.CreateCertificateTemplate(ctx, name, TeamID, uint(ValidCATypeID), "CN=$FLEET_VAR_HOST_UUID")
				require.NoError(t, err)
			})
		}
	})

	t.Run("Empty or whitespace-only subject name", func(t *testing.T) {
		whitespaceSubjectNames := []string{"", " ", "   \t\n  "}
		for _, subjectName := range whitespaceSubjectNames {
			_, err := svc.CreateCertificateTemplate(ctx, "my template", TeamID, uint(ValidCATypeID), subjectName)
			require.Error(t, err)
			require.Contains(t, err.Error(), "Certificate template subject name is required")
		}
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
	var nextTemplateID uint = 100

	ds.BatchUpsertCertificateTemplatesFunc = func(ctx context.Context, certificates []*fleet.CertificateTemplate) ([]uint, error) {
		createdCertificates = nil
		createdMap := make([]uint, 0, len(certificates))
		for _, cert := range certificates {
			createdCertificates = append(createdCertificates, *cert)
			createdMap = append(createdMap, cert.TeamID)
		}
		return createdMap, nil
	}

	ds.GetCertificateTemplatesByTeamIDFunc = func(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error) {
		var result []*fleet.CertificateTemplateResponseSummary
		for _, cert := range createdCertificates {
			if cert.TeamID == teamID {
				result = append(result, &fleet.CertificateTemplateResponseSummary{
					ID:   nextTemplateID,
					Name: cert.Name,
				})
				nextTemplateID++
			}
		}
		return result, nil, nil
	}

	ds.CreatePendingCertificateTemplatesForExistingHostsFunc = func(ctx context.Context, certificateTemplateID uint, teamID uint) (int64, error) {
		return 0, nil
	}

	ds.GetCertificateTemplateByTeamIDAndNameFunc = func(ctx context.Context, teamID uint, name string) (*fleet.CertificateTemplateResponse, error) {
		return &fleet.CertificateTemplateResponse{
			CertificateTemplateResponseSummary: fleet.CertificateTemplateResponseSummary{
				ID:   nextTemplateID,
				Name: name,
			},
			TeamID: teamID,
		}, nil
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

	t.Run("Empty name", func(t *testing.T) {
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   "",
				CertificateAuthorityId: 1,
				SubjectName:            "foo",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Certificate template name is required")
	})

	t.Run("Whitespace-only name", func(t *testing.T) {
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   "   ",
				CertificateAuthorityId: 1,
				SubjectName:            "foo",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Certificate template name is required")
	})

	t.Run("Name with invalid characters", func(t *testing.T) {
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   "template@name",
				CertificateAuthorityId: 1,
				SubjectName:            "foo",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Invalid certificate template name")
	})

	t.Run("Name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 256)
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   longName,
				CertificateAuthorityId: 1,
				SubjectName:            "foo",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Certificate template name is too long")
	})

	t.Run("Whitespace-only subject name", func(t *testing.T) {
		err := svc.ApplyCertificateTemplateSpecs(ctx, []*fleet.CertificateRequestSpec{
			{
				Name:                   "Template 2",
				CertificateAuthorityId: 1,
				SubjectName:            "   ",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Certificate template subject name is required")
		require.Contains(t, err.Error(), "Template 2")
	})
}

func TestResendHostCertificateTemplate(t *testing.T) {
	ds := new(mock.Store)
	opts := &TestServerOpts{}
	svc, ctx := newTestService(t, ds, nil, nil, opts)

	const (
		hostID       = uint(1)
		templateID   = uint(42)
		teamID       = uint(10)
		templateName = "My Cert"
	)

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id == hostID {
			tid := teamID
			return &fleet.Host{ID: id, TeamID: &tid}, nil
		}
		return nil, errors.New("host not found")
	}

	ds.GetCertificateTemplateByIdFunc = func(ctx context.Context, id uint) (*fleet.CertificateTemplateResponse, error) {
		return &fleet.CertificateTemplateResponse{
			CertificateTemplateResponseSummary: fleet.CertificateTemplateResponseSummary{
				ID:   id,
				Name: templateName,
			},
		}, nil
	}

	ds.GetCertificateTemplateByIdForHostFunc = func(ctx context.Context, id uint, hostUUID string) (*fleet.CertificateTemplateResponseForHost, error) {
		return &fleet.CertificateTemplateResponseForHost{
			CertificateTemplateResponse: fleet.CertificateTemplateResponse{
				CertificateTemplateResponseSummary: fleet.CertificateTemplateResponseSummary{
					ID:   id,
					Name: templateName,
				},
			},
			Status: fleet.CertificateTemplateDelivered,
		}, nil
	}

	t.Run("succeeds and creates activity", func(t *testing.T) {
		ds.ResendHostCertificateTemplateFunc = func(ctx context.Context, hID uint, tID uint) error {
			require.Equal(t, hostID, hID)
			require.Equal(t, templateID, tID)
			return nil
		}

		var capturedActivity fleet.ActivityTypeResentCertificate
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
			act, ok := activity.(fleet.ActivityTypeResentCertificate)
			require.True(t, ok, "expected ActivityTypeResentCertificate, got %T", activity)
			capturedActivity = act
			return nil
		}

		err := svc.ResendHostCertificateTemplate(ctx, hostID, templateID)
		require.NoError(t, err)
		require.True(t, ds.ResendHostCertificateTemplateFuncInvoked)
		require.True(t, opts.ActivityMock.NewActivityFuncInvoked)
		require.Equal(t, hostID, capturedActivity.HostID)
		require.Equal(t, templateID, capturedActivity.CertificateTemplateID)
		require.Equal(t, templateName, capturedActivity.CertificateName)

		ds.ResendHostCertificateTemplateFuncInvoked = false
		opts.ActivityMock.NewActivityFuncInvoked = false
	})

	t.Run("returns error when host not found", func(t *testing.T) {
		err := svc.ResendHostCertificateTemplate(ctx, 99999, templateID)
		require.Error(t, err)
		require.False(t, opts.ActivityMock.NewActivityFuncInvoked)
	})

	t.Run("returns error when datastore fails", func(t *testing.T) {
		ds.ResendHostCertificateTemplateFunc = func(ctx context.Context, hID uint, tID uint) error {
			return errors.New("db error")
		}

		err := svc.ResendHostCertificateTemplate(ctx, hostID, templateID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "db error")
		require.False(t, opts.ActivityMock.NewActivityFuncInvoked)
	})

	t.Run("returns 400 when template is pending for host", func(t *testing.T) {
		ds.GetCertificateTemplateByIdForHostFunc = func(ctx context.Context, id uint, hostUUID string) (*fleet.CertificateTemplateResponseForHost, error) {
			return &fleet.CertificateTemplateResponseForHost{
				CertificateTemplateResponse: fleet.CertificateTemplateResponse{
					CertificateTemplateResponseSummary: fleet.CertificateTemplateResponseSummary{
						ID:   id,
						Name: templateName,
					},
				},
				Status: fleet.CertificateTemplatePending,
			}, nil
		}
		ds.ResendHostCertificateTemplateFuncInvoked = false

		err := svc.ResendHostCertificateTemplate(ctx, hostID, templateID)
		require.Error(t, err)

		var umErr interface{ StatusCode() int }
		require.ErrorAs(t, err, &umErr)
		require.Equal(t, 400, umErr.StatusCode())
		require.False(t, ds.ResendHostCertificateTemplateFuncInvoked)
		require.False(t, opts.ActivityMock.NewActivityFuncInvoked)
	})
}
