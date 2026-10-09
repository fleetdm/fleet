package mdm

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	euaPendingKeyPrefix = "eua_pending:"
	// maxEUAPendingHostUUIDLen matches the hosts.uuid column width, so a caller
	// cannot turn an unbounded request field into an unbounded Redis key.
	maxEUAPendingHostUUIDLen = 255
	// euaPendingTTL garbage-collects records for devices that never come back.
	euaPendingTTL = 24 * time.Hour
)

// A record under this key says Fleet answered an Orbit enroll request for that
// hardware UUID with END_USER_AUTH_REQUIRED and is therefore expecting the
// device's end user to sign in.
func euaPendingKey(hostUUID string) string {
	return euaPendingKeyPrefix + hostUUID
}

// validEUAPendingHostUUID rejects what must never become a key. An empty UUID
// would make a single shared record stand for every device, and an over-long one
// is a request field nothing upstream has bounded.
func validEUAPendingHostUUID(hostUUID string) bool {
	return hostUUID != "" && len(hostUUID) <= maxEUAPendingHostUUIDLen
}

// setEUAPendingExpiry writes the record's deadline. Recording puts it in the
// future and clearing puts it at now, which is the only difference between the
// two operations.
func setEUAPendingExpiry(
	ctx context.Context, kv fleet.KeyValueStore, hostUUID string, expiresAt time.Time, ttl time.Duration,
) error {
	if kv == nil {
		return ctxerr.New(ctx, "end user auth prompt store not configured")
	}
	if !validEUAPendingHostUUID(hostUUID) {
		return ctxerr.New(ctx, "invalid host uuid for end user auth prompt")
	}
	if err := kv.Set(ctx, euaPendingKey(hostUUID), expiresAt.Format(time.RFC3339Nano), ttl); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user auth prompt")
	}
	return nil
}

// RecordEndUserAuthPrompt stores the prompt for hostUUID. Callers record on the
// enroll response path, so a device that is told to authenticate is always the
// one that can start the SSO flow.
func RecordEndUserAuthPrompt(ctx context.Context, kv fleet.KeyValueStore, hostUUID string, now time.Time) error {
	return setEUAPendingExpiry(ctx, kv, hostUUID, now.Add(euaPendingTTL), euaPendingTTL)
}

// ClearEndUserAuthPrompt ends the prompt once the host has enrolled.
func ClearEndUserAuthPrompt(ctx context.Context, kv fleet.KeyValueStore, hostUUID string) error {
	return setEUAPendingExpiry(ctx, kv, hostUUID, time.Time{}, time.Second)
}

// HasEndUserAuthPrompt reports whether Fleet is waiting on this device's end
// user. An error means the store failed and the caller must fail closed.
func HasEndUserAuthPrompt(ctx context.Context, kv fleet.KeyValueStore, hostUUID string, now time.Time) (bool, error) {
	if kv == nil {
		return false, ctxerr.New(ctx, "end user auth prompt store not configured")
	}
	if !validEUAPendingHostUUID(hostUUID) {
		return false, nil
	}
	val, err := kv.Get(ctx, euaPendingKey(hostUUID))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "get end user auth prompt")
	}
	if val == nil {
		return false, nil
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, *val)
	if err != nil {
		return false, nil
	}
	return now.Before(expiresAt), nil
}
