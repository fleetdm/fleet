package async

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
)

const (
	labelMembershipHostKey     = "label_membership:{%d}"
	labelMembershipReportedKey = "label_membership_reported:{%d}"
)

type Task struct {
	Datastore fleet.Datastore
	Pool      fleet.RedisPool
	// AsyncEnabled indicates if async processing is enabled in the
	// configuration. Note that Pool can be nil if this is false.
	AsyncEnabled bool // TODO: should this be read in a different way, more dynamically, if config changes while fleet is running?
}

func (t *Task) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
	if !t.AsyncEnabled {
		host.LabelUpdatedAt = ts
		return t.Datastore.RecordLabelQueryExecutions(ctx, host, results, ts)
	}

	conn := t.Pool.ConfigureDoer(t.Pool.Get())
	defer conn.Close()

	keySet := fmt.Sprintf(labelMembershipHostKey, host.ID)
	keyTs := fmt.Sprintf(labelMembershipReportedKey, host.ID)
	// TODO: prepare results, store as -1 for delete, +1 for insert
	if _, err := conn.Do("ZADD", keySet); err != nil {
		return err
	}
	// ignore error if only the timestamp setting fails
	// TODO: could be improved by using a Lua script, atomically run both commands
	_, _ = conn.Do("SET", keyTs, ts.Unix())
	return nil
}

func (t *Task) GetHostLabelReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	if t.AsyncEnabled {
		conn := t.Pool.ConfigureDoer(t.Pool.Get())
		defer conn.Close()

		key := fmt.Sprintf(labelMembershipReportedKey, host.ID)
		epoch, err := redis.Int64(conn.Do("GET", key))
		if err == nil {
			return time.Unix(epoch, 0)
		}
	}
	return host.LabelUpdatedAt
}
