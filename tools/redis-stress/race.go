package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// raceStats accumulates per-run counts for the race-detection mode.
type raceStats struct {
	sets       atomic.Int64
	setErrs    atomic.Int64
	gets       atomic.Int64
	getErrs    atomic.Int64
	getNilRace atomic.Int64 // GET returned nil after a successful SET — the bug we're chasing.
	getStale   atomic.Int64 // GET returned a value, but not the one we just set (would be very odd).
}

func runRace(args []string) {
	fs := flag.NewFlagSet("race", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "redis-stress race — tight SET-then-GET race detection.\n\n")
		fmt.Fprintf(fs.Output(), "Each worker repeatedly does:\n")
		fmt.Fprintf(fs.Output(), "  conn1 := pool.Get(); conn1.Do(\"SET\", k, v, \"PX\", ttl); conn1.Close()\n")
		fmt.Fprintf(fs.Output(), "  conn2 := pool.Get(); conn2.Do(\"GET\", k);                conn2.Close()\n")
		fmt.Fprintf(fs.Output(), "and reports any GET that returns nil immediately after a successful SET.\n")
		fmt.Fprintf(fs.Output(), "This mirrors how Fleet's RedisKeyValue.Set/.Get use the connection pool.\n\n")
		fmt.Fprintf(fs.Output(), "FLAGS:\n")
		fs.PrintDefaults()
	}
	addr := fs.String("addr", "127.0.0.1:7001", "Redis cluster startup node")
	workers := fs.Int("workers", 50, "Concurrent SET-then-GET workers")
	iterations := fs.Int("iterations", 1000, "Iterations per worker")
	ttl := fs.Duration("ttl", 4*time.Minute, "PX expiration on SET")
	keyPrefix := fs.String("key-prefix", "stress_race_", "Key prefix")
	explicitReadOnly := fs.Bool("explicit-readonly", false,
		"Call redis.ReadOnlyConn on the GET conn (routes reads to replicas in cluster mode); "+
			"set this together with -cluster-read-from-replica to test replica-lag scenarios")
	followRedirs := fs.Bool("cluster-follow-redirects", true, "ClusterFollowRedirections")
	readReplica := fs.Bool("cluster-read-from-replica", true, "ClusterReadFromReplica")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *workers < 1 {
		log.Fatalf("workers must be >= 1, got %d", *workers)
	}
	if *iterations < 1 {
		log.Fatalf("iterations must be >= 1, got %d", *iterations)
	}

	pool, err := redis.NewPool(redis.PoolConfig{
		Server:                    *addr,
		UseTLS:                    false,
		ClusterFollowRedirections: *followRedirs,
		ClusterReadFromReplica:    *readReplica,
		ConnTimeout:               5 * time.Second,
		ReadTimeout:               5 * time.Second,
		WriteTimeout:              5 * time.Second,
		MaxIdleConns:              *workers * 2,
		MaxOpenConns:              *workers * 4,
	})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	log.Printf("race mode: addr=%s read_from_replica=%v follow_redirects=%v explicit_readonly=%v",
		*addr, *readReplica, *followRedirs, *explicitReadOnly)
	log.Printf("workers=%d iterations=%d ttl=%s", *workers, *iterations, *ttl)

	var s raceStats
	var wg sync.WaitGroup
	start := time.Now()

	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < *iterations; i++ {
				key := fmt.Sprintf("%sw%d_i%d", *keyPrefix, id, i)
				expected := fmt.Sprintf("v_%d_%d", id, i)
				if err := raceOnce(pool, key, expected, ttl.Milliseconds(), *explicitReadOnly, &s); err != nil {
					log.Printf("[w%d.i%d] %v", id, i, err)
				}
			}
		}(w)
	}
	wg.Wait()

	elapsed := time.Since(start)
	total := s.sets.Load() + s.gets.Load()

	fmt.Println()
	fmt.Println("================ summary ================")
	fmt.Printf("elapsed:           %s\n", elapsed)
	fmt.Printf("sets:              %d (errors %d)\n", s.sets.Load(), s.setErrs.Load())
	fmt.Printf("gets:              %d (errors %d)\n", s.gets.Load(), s.getErrs.Load())
	fmt.Printf("nil-after-set:     %d  ← the bug\n", s.getNilRace.Load())
	fmt.Printf("stale-after-set:   %d\n", s.getStale.Load())
	fmt.Printf("ops/sec:           %.1f\n", float64(total)/elapsed.Seconds())

	if s.getNilRace.Load() > 0 {
		fmt.Println()
		fmt.Println("RESULT: SET-visibility race observed.")
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("RESULT: no race observed under these conditions.")
}

// raceOnce mimics RedisKeyValue.Set immediately followed by RedisKeyValue.Get,
// each on a fresh pool connection. If explicitReadOnly is true, the GET conn
// is converted via redis.ReadOnlyConn — only effective in cluster mode with
// ClusterReadFromReplica=true on the pool.
func raceOnce(pool fleet.RedisPool, key, expected string, ttlMs int64, explicitReadOnly bool, s *raceStats) error {
	// SET
	{
		conn := redis.ConfigureDoer(pool, pool.Get())
		_, err := redigo.String(conn.Do("SET", key, expected, "PX", ttlMs))
		conn.Close()
		if err != nil {
			s.setErrs.Add(1)
			return fmt.Errorf("set: %w", err)
		}
		s.sets.Add(1)
	}

	// GET (optionally via a read-only conn that may land on a replica)
	{
		conn := redis.ConfigureDoer(pool, pool.Get())
		if explicitReadOnly {
			conn = redis.ReadOnlyConn(pool, conn)
		}
		got, err := redigo.String(conn.Do("GET", key))
		conn.Close()
		if errors.Is(err, redigo.ErrNil) {
			s.getNilRace.Add(1)
			s.gets.Add(1)
			fmt.Printf("RACE key=%s expected=%s got=<nil>\n", key, expected)
			return nil
		}
		if err != nil {
			s.getErrs.Add(1)
			return fmt.Errorf("get: %w", err)
		}
		s.gets.Add(1)
		if got != expected {
			s.getStale.Add(1)
			fmt.Printf("STALE key=%s expected=%s got=%s\n", key, expected, got)
		}
	}

	// Best-effort cleanup so we don't pile up keys with the TTL.
	{
		conn := redis.ConfigureDoer(pool, pool.Get())
		_, _ = conn.Do("DEL", key)
		conn.Close()
	}
	return nil
}
