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
	collectedTooLongAgo := now.Add(-BitLockerPINResultTimeout - time.Second)
	request := func(status BitLockerPINRequestStatus, createdAt, updatedAt time.Time) *HostBitLockerPINRequest {
		return &HostBitLockerPINRequest{Status: status, CreatedAt: createdAt, UpdatedAt: updatedAt}
	}

	for _, tc := range []struct {
		name string
		req  *HostBitLockerPINRequest
		want bool
	}{
		{"nil request", nil, false},
		{"fresh pending", request(BitLockerPINRequestPending, now, now), false},
		{"stale pending", request(BitLockerPINRequestPending, stale, stale), true},
		// A delivered request is timed from collection, not submission, so a PIN collected late still gets the full timeout.
		{"delivered within the result timeout", request(BitLockerPINRequestDelivered, stale, now), false},
		{"delivered past the result timeout", request(BitLockerPINRequestDelivered, collectedTooLongAgo, collectedTooLongAgo), true},
		{"stale set", request(BitLockerPINRequestSet, collectedTooLongAgo, collectedTooLongAgo), false},
		{"stale failed", request(BitLockerPINRequestFailed, collectedTooLongAgo, collectedTooLongAgo), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.req.Expired(now))
		})
	}
}
