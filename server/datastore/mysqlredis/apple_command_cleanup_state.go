package mysqlredis

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// appleCommandCleanupStateKey is the Redis key holding the Apple MDM command
// cleanup cron's cursors, as JSON.
const appleCommandCleanupStateKey = "mdm:apple:command_cleanup_state"

var _ fleet.MDMAppleCommandCleanupStateStore = (*Datastore)(nil)

// GetMDMAppleCommandCleanupState returns the Apple MDM command cleanup cron's
// persisted cursors, or nil when none are stored.
//
// No TTL: loss of this key is harmless — every scan restarts from the oldest
// rows and re-examining already-swept ranges is idempotent. Undecodable state
// is treated the same way rather than wedging the cron on a poisoned key.
func (d *Datastore) GetMDMAppleCommandCleanupState(ctx context.Context) (*fleet.MDMAppleCommandCleanupState, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	raw, err := redigo.Bytes(conn.Do("GET", appleCommandCleanupStateKey))
	switch {
	case err == nil:
		var state fleet.MDMAppleCommandCleanupState
		if err := json.Unmarshal(raw, &state); err != nil {
			// complete the self-heal: drop the poisoned key so it doesn't
			// linger for the next read or a confused operator
			d.logger.WarnContext(ctx, "dropping undecodable Apple MDM command cleanup state key, restarting scans",
				"key", appleCommandCleanupStateKey, "err", err)
			_, _ = conn.Do("DEL", appleCommandCleanupStateKey)
			return nil, nil
		}
		return &state, nil
	case errors.Is(err, redigo.ErrNil):
		return nil, nil
	default:
		return nil, ctxerr.Wrap(ctx, err, "get apple MDM command cleanup state")
	}
}

// SetMDMAppleCommandCleanupState persists the Apple MDM command cleanup cron's
// cursors. A nil state deletes the key.
func (d *Datastore) SetMDMAppleCommandCleanupState(ctx context.Context, state *fleet.MDMAppleCommandCleanupState) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if state == nil {
		if _, err := conn.Do("DEL", appleCommandCleanupStateKey); err != nil {
			return ctxerr.Wrap(ctx, err, "delete apple MDM command cleanup state")
		}
		return nil
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshal apple MDM command cleanup state")
	}
	if _, err := conn.Do("SET", appleCommandCleanupStateKey, raw); err != nil {
		return ctxerr.Wrap(ctx, err, "set apple MDM command cleanup state")
	}
	return nil
}
