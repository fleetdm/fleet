package async

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

type collector struct {
	// immutable after creation
	name         string
	pool         fleet.RedisPool
	ds           fleet.Datastore
	execInterval time.Duration
	jitterPct    int
	lockTimeout  time.Duration
	handler      func(context.Context, fleet.Datastore, fleet.RedisPool, *collectorExecStats) error
	errHandler   func(string, error)

	// mutable, must be protected by mutex
	mu    sync.RWMutex
	stats collectorStats
}

type collectorStats struct {
	SkipCount     int
	ExecCount     int
	FailuresCount int

	MinExecDuration  time.Duration
	MaxExecDuration  time.Duration
	LastExecDuration time.Duration

	MinExecKeys  int
	MaxExecKeys  int
	LastExecKeys int

	MinExecItems  int
	MaxExecItems  int
	LastExecItems int

	MinExecInserts  int
	MaxExecInserts  int
	LastExecInserts int

	MinExecUpdates  int
	MaxExecUpdates  int
	LastExecUpdates int

	MinExecDeletes  int
	MaxExecDeletes  int
	LastExecDeletes int

	MinExecRedisCmds  int
	MaxExecRedisCmds  int
	LastExecRedisCmds int
}

type collectorExecStats struct {
	Duration  time.Duration
	Keys      int
	Items     int
	Inserts   int
	Updates   int
	Deletes   int
	RedisCmds int // does not include scan keys iteration commands
	Failed    bool
}

func (c *collector) Start(ctx context.Context) {
	for {
		wait := c.nextRunAfter()
		select {
		case <-time.After(wait):
			c.exec(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *collector) exec(ctx context.Context) {
	keyLock := fmt.Sprintf(collectorLockKey, c.name)
	conn := redis.ConfigureDoer(c.pool, c.pool.Get())
	defer conn.Close()

	if _, err := redigo.String(conn.Do("SET", keyLock, 1, "NX", "EX", int(c.lockTimeout.Seconds()))); err != nil {
		var failed bool
		// either redis failure or this collector didn't acquire the lock
		if !errors.Is(err, redigo.ErrNil) {
			failed = true
			if c.errHandler != nil {
				c.errHandler(c.name, err)
			}
		}
		c.addSkipStats(failed)
		return
	}
	defer conn.Do("DEL", keyLock)

	// at this point, the lock has been acquired, execute the collector handler
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.lockTimeout.Seconds())*time.Second)
	defer cancel()

	var stats collectorExecStats
	start := time.Now()
	if err := c.handler(ctx, c.ds, c.pool, &stats); err != nil {
		stats.Failed = true
		if c.errHandler != nil {
			c.errHandler(c.name, err)
		}
	}
	stats.Duration = time.Since(start)
	c.addStats(&stats)
}

func (c *collector) ReadStats() collectorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *collector) addStats(stats *collectorExecStats) {
	minMaxLastInt := func(min, max, last *int, val int) {
		*last = val
		if val > *max {
			*max = val
		}
		if val < *min || (val > 0 && (*min == 0)) {
			*min = val
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.ExecCount++
	if stats.Failed {
		c.stats.FailuresCount++
	}

	minMaxLastInt(&c.stats.MinExecItems, &c.stats.MaxExecItems, &c.stats.LastExecItems, stats.Items)
	minMaxLastInt(&c.stats.MinExecKeys, &c.stats.MaxExecKeys, &c.stats.LastExecKeys, stats.Keys)
	minMaxLastInt(&c.stats.MinExecInserts, &c.stats.MaxExecInserts, &c.stats.LastExecInserts, stats.Inserts)
	minMaxLastInt(&c.stats.MinExecUpdates, &c.stats.MaxExecUpdates, &c.stats.LastExecUpdates, stats.Updates)
	minMaxLastInt(&c.stats.MinExecDeletes, &c.stats.MaxExecDeletes, &c.stats.LastExecDeletes, stats.Deletes)
	minMaxLastInt(&c.stats.MinExecRedisCmds, &c.stats.MaxExecRedisCmds, &c.stats.LastExecRedisCmds, stats.RedisCmds)

	c.stats.LastExecDuration = stats.Duration
	if stats.Duration > c.stats.MaxExecDuration {
		c.stats.MaxExecDuration = stats.Duration
	}
	if stats.Duration < c.stats.MinExecDuration || (stats.Duration > 0 && c.stats.MinExecDuration == 0) {
		c.stats.MinExecDuration = stats.Duration
	}
}

func (c *collector) addSkipStats(failed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.SkipCount++
	if failed {
		c.stats.FailuresCount++
	}
}

func (c *collector) nextRunAfter() time.Duration {
	var jitter time.Duration
	if c.jitterPct > 0 {
		maxJitter := time.Duration(c.jitterPct) * c.execInterval / time.Duration(100.0)
		randDuration, err := rand.Int(rand.Reader, big.NewInt(int64(maxJitter)))
		if err == nil {
			jitter = time.Duration(randDuration.Int64())
		}
	}

	// randomize running after or before the jitter based on even/odd
	if jitter%2 == 0 {
		return c.execInterval - jitter
	}
	return c.execInterval + jitter
}
