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

// apnsSweepStateKey is the Redis key holding the APNs sweep cron's pass
// state (keyset cursor + the batch size computed at pass start), as JSON.
const apnsSweepStateKey = "mdm:apple:apns_sweep_state"

// GetMDMAppleAPNsSweepState returns the APNs sweep cron's persisted pass
// state, or nil when no pass is in progress.
//
// No TTL: loss of this key is harmless — the cron starts a fresh pass from
// the beginning with a recomputed batch size, and re-pushing an enrollment
// is a contentless, idempotent nudge. Undecodable state is treated the same
// way rather than wedging the cron on a poisoned key.
func (d *Datastore) GetMDMAppleAPNsSweepState(ctx context.Context) (*fleet.MDMAppleAPNsSweepState, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	raw, err := redigo.Bytes(conn.Do("GET", apnsSweepStateKey))
	switch {
	case err == nil:
		var state fleet.MDMAppleAPNsSweepState
		if err := json.Unmarshal(raw, &state); err != nil {
			// complete the self-heal: drop the poisoned key so it doesn't
			// linger for the next read or a confused operator
			d.logger.WarnContext(ctx, "dropping undecodable APNs sweep state key, starting a fresh pass",
				"key", apnsSweepStateKey, "err", err)
			_, _ = conn.Do("DEL", apnsSweepStateKey)
			return nil, nil
		}
		return &state, nil
	case errors.Is(err, redigo.ErrNil):
		return nil, nil
	default:
		return nil, ctxerr.Wrap(ctx, err, "get apple MDM APNs sweep state")
	}
}

// SetMDMAppleAPNsSweepState persists the APNs sweep cron's pass state. A nil
// state deletes the key (pass complete; the next tick starts a new pass).
func (d *Datastore) SetMDMAppleAPNsSweepState(ctx context.Context, state *fleet.MDMAppleAPNsSweepState) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if state == nil {
		if _, err := conn.Do("DEL", apnsSweepStateKey); err != nil {
			return ctxerr.Wrap(ctx, err, "delete apple MDM APNs sweep state")
		}
		return nil
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshal apple MDM APNs sweep state")
	}
	if _, err := conn.Do("SET", apnsSweepStateKey, raw); err != nil {
		return ctxerr.Wrap(ctx, err, "set apple MDM APNs sweep state")
	}
	return nil
}
