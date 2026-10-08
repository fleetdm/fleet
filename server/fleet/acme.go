package fleet

import "context"

// ACMEWriteService is the subset of the ACME service module service
// used by the legacy service layer for write operations.
type ACMEWriteService interface {
	// NewACMEEnrollment creates a new enrollment in the acme_enrollments table for the host identifier (the
	// hardware serial) and returns its path_identifier. The purpose is written into the issued certificate's
	// binding. enrollmentID is the enrollment's device channel ID, required for acme_renewal and nil for acme.
	NewACMEEnrollment(ctx context.Context, hostIdentifier string, purpose AppleMDMCertPurpose, enrollmentID *string) (string, error)
}
