package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
)

func TestGetDeviceCertificateTemplateErrors(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	host := &fleet.Host{UUID: "host-uuid-123", ID: 1, TeamID: nil}
	ctx = test.HostContext(ctx, host)

	ds.GetCertificateTemplateByIdForHostFunc = func(ctxCtx context.Context, certificateTemplateID uint, hostUUID string) (*fleet.CertificateTemplateResponseForHost, error) {
		return &fleet.CertificateTemplateResponseForHost{
			Status: fleet.CertificateTemplatePending,
		}, nil
	}

	ds.UpsertCertificateStatusFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint, status fleet.MDMDeliveryStatus, detail *string) error {
		return nil
	}

	cases := []struct {
		name           string
		templateStatus fleet.CertificateTemplateStatus
		err            error
	}{
		{
			name:           "delivered status",
			templateStatus: fleet.CertificateTemplateDelivered,
			err:            nil,
		},
		{
			name:           "failed status",
			templateStatus: fleet.CertificateTemplateFailed,
			err:            newNotFoundError(),
		},
		{
			name:           "verified status",
			templateStatus: fleet.CertificateTemplateVerified,
			err:            newNotFoundError(),
		},
		{
			name:           "pending status",
			templateStatus: fleet.CertificateTemplatePending,
			err:            newNotFoundError(),
		},
		{
			name:           "delivering status",
			templateStatus: fleet.CertificateTemplateDelivering,
			err:            newNotFoundError(),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds.GetCertificateTemplateByIdForHostFunc = func(ctxCtx context.Context, certificateTemplateID uint, hostUUID string) (*fleet.CertificateTemplateResponseForHost, error) {
				return &fleet.CertificateTemplateResponseForHost{
					Status: c.templateStatus,
					CertificateTemplateResponse: fleet.CertificateTemplateResponse{
						TeamID: 0,
					},
				}, nil
			}

			_, err := svc.GetDeviceCertificateTemplate(ctx, 1)
			if c.err != nil {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
