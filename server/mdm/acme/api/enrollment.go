package api

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// EnrollmentService stores records in the acme_enrollments table.
type EnrollmentService interface {
	// NewACMEEnrollment creates a new enrollment in the acme_enrollments table with the specified
	// host identifier and purpose, and returns a new path_identifier for the created row. enrollmentID is
	// required for acme_renewal and must be nil for acme.
	NewACMEEnrollment(ctx context.Context, hostIdentifier string, purpose fleet.AppleMDMCertPurpose, enrollmentID *string) (string, error)
}
