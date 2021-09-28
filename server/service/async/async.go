package async

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
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

	keySet := fmt.Sprintf(labelMembershipHostKey, host.ID)
	keyTs := fmt.Sprintf(labelMembershipReportedKey, host.ID)

	script := redigo.NewScript(2, `
    redis.call('ZADD', KEYS[1], unpack(ARGV, 2))
    return redis.call('SET', KEYS[2], ARGV[1])
  `)

	// convert results to ZADD arguments, store as -1 for delete, +1 for insert
	args := make(redigo.Args, 0, 3+(len(results)*2))
	args = args.Add(keySet, keyTs, ts.Unix())
	for k, v := range results {
		score := -1
		if v != nil && *v {
			score = 1
		}
		args = args.Add(score, k)
	}

	conn := t.Pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.Pool, conn, keySet, keyTs); err != nil {
		return errors.Wrap(err, "bind redis connection")
	}

	if _, err := script.Do(conn, args...); err != nil {
		return err
	}
	return nil
}

func (t *Task) GetHostLabelReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	if t.AsyncEnabled {
		conn := t.Pool.ConfigureDoer(t.Pool.Get())
		defer conn.Close()

		key := fmt.Sprintf(labelMembershipReportedKey, host.ID)
		epoch, err := redigo.Int64(conn.Do("GET", key))
		if err == nil {
			return time.Unix(epoch, 0)
		}
	}
	return host.LabelUpdatedAt
}
