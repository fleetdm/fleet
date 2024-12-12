package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
)

func TestLinuxHostDiskEncryptionStatus(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	actionRequired := fleet.DiskEncryptionActionRequired
	verified := fleet.DiskEncryptionVerified
	failed := fleet.DiskEncryptionFailed

	testcases := []struct {
		name              string
		host              fleet.Host
		keyExists         bool
		clientErrorExists bool
		status            fleet.HostMDMDiskEncryption
		notFound          bool
	}{
		{
			name:              "no key",
			host:              fleet.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         false,
			clientErrorExists: false,
			status: fleet.HostMDMDiskEncryption{
				Status: &actionRequired,
			},
		},
		{
			name:              "key exists",
			host:              fleet.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         true,
			clientErrorExists: false,
			status: fleet.HostMDMDiskEncryption{
				Status: &verified,
			},
		},
		{
			name:              "key exists && client error",
			host:              fleet.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         true,
			clientErrorExists: true,
			status: fleet.HostMDMDiskEncryption{
				Status: &failed,
				Detail: "client error",
			},
		},
		{
			name:              "no key && client error",
			host:              fleet.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         false,
			clientErrorExists: true,
			status: fleet.HostMDMDiskEncryption{
				Status: &failed,
				Detail: "client error",
			},
		},
		{
			name:              "key not found",
			host:              fleet.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         false,
			clientErrorExists: false,
			status: fleet.HostMDMDiskEncryption{
				Status: &actionRequired,
			},
			notFound: true,
		},
		{
			name:   "unsupported platform",
			host:   fleet.Host{ID: 1, Platform: "amzn"},
			status: fleet.HostMDMDiskEncryption{},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			ds.GetHostDiskEncryptionKeyFunc = func(ctx context.Context, hostID uint) (*fleet.HostDiskEncryptionKey, error) {
				var encrypted string
				if tt.keyExists {
					encrypted = "encrypted"
				}

				var clientError string
				if tt.clientErrorExists {
					clientError = "client error"
				}

				var nfe notFoundError
				if tt.notFound {
					return nil, &nfe
				}

				return &fleet.HostDiskEncryptionKey{
					HostID:          hostID,
					Base64Encrypted: encrypted,
					Decryptable:     ptr.Bool(true),
					UpdatedAt:       time.Now(),
					ClientError:     clientError,
				}, nil
			}

			status, err := svc.LinuxHostDiskEncryptionStatus(ctx, tt.host)
			assert.Nil(t, err)

			assert.Equal(t, tt.status, status)
		})
	}
}
