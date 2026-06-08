package failing

import (
	"context"
	"errors"
	"io"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// FailingOrgLogoStore is an implementation of fleet.OrgLogoStore that fails all
// operations. It is used when S3 is not configured and the local filesystem
// store could not be set up (e.g. a read-only root filesystem).
//
// It does not embed commonFailingStore because fleet.OrgLogoStore is keyed on
// fleet.OrgLogoMode (not string) and exposes Delete rather than Cleanup/Sign.
type FailingOrgLogoStore struct{}

var _ fleet.OrgLogoStore = (*FailingOrgLogoStore)(nil)

var ErrOrgLogoStoreNotConfigured = errors.New("org logo store not properly configured")

func NewFailingOrgLogoStore() *FailingOrgLogoStore {
	return &FailingOrgLogoStore{}
}

func (FailingOrgLogoStore) Put(_ context.Context, _ fleet.OrgLogoMode, _ io.ReadSeeker) error {
	return ErrOrgLogoStoreNotConfigured
}

func (FailingOrgLogoStore) Get(_ context.Context, _ fleet.OrgLogoMode) (io.ReadCloser, int64, error) {
	return nil, 0, ErrOrgLogoStoreNotConfigured
}

func (FailingOrgLogoStore) Delete(_ context.Context, _ fleet.OrgLogoMode) error {
	return ErrOrgLogoStoreNotConfigured
}

func (FailingOrgLogoStore) Exists(_ context.Context, _ fleet.OrgLogoMode) (bool, error) {
	return false, ErrOrgLogoStoreNotConfigured
}
