package fleet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBitLockerPINRequestExpired(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stale := now.Add(-BitLockerPINRequestTTL - time.Second)
	request := func(status BitLockerPINRequestStatus, createdAt time.Time) *HostBitLockerPINRequest {
		return &HostBitLockerPINRequest{Status: status, CreatedAt: createdAt}
	}

	for _, tc := range []struct {
		name string
		req  *HostBitLockerPINRequest
		want bool
	}{
		{"nil request", nil, false},
		{"fresh pending", request(BitLockerPINRequestPending, now), false},
		{"stale pending", request(BitLockerPINRequestPending, stale), true},
		// Once the agent has the PIN, the outcome is its to report, so age stops mattering.
		{"stale delivered", request(BitLockerPINRequestDelivered, stale), false},
		{"stale set", request(BitLockerPINRequestSet, stale), false},
		{"stale failed", request(BitLockerPINRequestFailed, stale), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.req.Expired(now))
		})
	}
}
