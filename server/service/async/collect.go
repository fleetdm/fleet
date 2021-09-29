package async

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

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
}

type collectorExecStats struct {
	Duration time.Duration
	Keys     int
	Items    int
	Failed   bool
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
	conn := c.pool.ConfigureDoer(c.pool.Get())
	defer conn.Close()

	if _, err := redigo.String(conn.Do("SET", keyLock, 1, "NX", "EX", int(c.lockTimeout/time.Second))); err != nil {
		// either redis failure or this collector didn't acquire the lock
		if !errors.Is(err, redigo.ErrNil) && c.errHandler != nil {
			c.errHandler(c.name, err)
		}
		return
	}
	defer conn.Do("DEL", keyLock)

	// at this point, the lock has been acquired, execute the collector handler
	ctx, cancel := context.WithTimeout(ctx, c.lockTimeout/time.Second)
	defer cancel()

	var stats collectorExecStats
	start := time.Now()
	if err := c.handler(ctx, c.ds, c.pool, &stats); err != nil && c.errHandler != nil {
		stats.Failed = true
		c.errHandler(c.name, err)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.ExecCount++
	if stats.Failed {
		c.stats.FailuresCount++
	}

	c.stats.LastExecItems = stats.Items
	if stats.Items > c.stats.MaxExecItems {
		c.stats.MaxExecItems = stats.Items
	}
	if stats.Items < c.stats.MinExecItems || (stats.Items > 0 && c.stats.MinExecItems == 0) {
		c.stats.MinExecItems = stats.Items
	}

	c.stats.LastExecKeys = stats.Keys
	if stats.Keys > c.stats.MaxExecKeys {
		c.stats.MaxExecKeys = stats.Keys
	}
	if stats.Keys < c.stats.MinExecKeys || (stats.Keys > 0 && c.stats.MinExecKeys == 0) {
		c.stats.MinExecKeys = stats.Keys
	}

	c.stats.LastExecDuration = stats.Duration
	if stats.Duration > c.stats.MaxExecDuration {
		c.stats.MaxExecDuration = stats.Duration
	}
	if stats.Duration < c.stats.MinExecDuration || (stats.Duration > 0 && c.stats.MinExecDuration == 0) {
		c.stats.MinExecDuration = stats.Duration
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
