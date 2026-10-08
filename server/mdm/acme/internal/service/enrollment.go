package service

import (
	"context"
	"fmt"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (s *Service) NewACMEEnrollment(ctx context.Context, hostIdentifier string, purpose fleet.AppleMDMCertPurpose, enrollmentID *string) (string, error) {
	// skipauth: No authorization check needed; caller is authenticated via DEP device identity.
	if az, ok := authz_ctx.FromContext(ctx); ok {
		az.SetChecked()
	}

	switch purpose {
	case fleet.AppleMDMCertPurposeACME:
		if enrollmentID != nil {
			return "", ctxerr.New(ctx, "an acme enrollment can't carry an enrollment ID")
		}
	case fleet.AppleMDMCertPurposeACMERenewal:
		if enrollmentID == nil || *enrollmentID == "" {
			return "", ctxerr.New(ctx, "an acme_renewal enrollment requires an enrollment ID")
		}
	default:
		return "", ctxerr.New(ctx, fmt.Sprintf("unsupported ACME enrollment purpose: %s", purpose))
	}

	return s.store.NewEnrollment(ctx, hostIdentifier, string(purpose), enrollmentID)
}
