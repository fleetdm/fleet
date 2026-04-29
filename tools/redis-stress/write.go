package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

func runWrite(args []string) {
	fs := flag.NewFlagSet("write", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "redis-stress write — steady SET-only load against Redis or a Redis cluster.\n\n")
		fmt.Fprintf(fs.Output(), "FLAGS:\n")
		fs.PrintDefaults()
	}
	addr := fs.String("addr", "127.0.0.1:6379", "Redis address (cluster startup node OK; cluster mode is auto-detected)")
	workers := fs.Int("workers", 1, "Concurrent SET workers")
	rate := fs.Float64("rate", 1.0, "SETs per worker per second")
	duration := fs.Duration("duration", 10*time.Minute, "Total run time")
	wait := fs.Duration("wait", 0, "Alias for -duration (legacy; kept for backward compatibility with the original tool)")
	keyPrefix := fs.String("key-prefix", "stress_write_", "Key prefix")
	keyTTL := fs.Duration("key-ttl", 10*time.Minute, "Per-key expiration")
	indexStart := fs.Int("index-start", 0, "Starting value of each worker's per-key counter")
	debug := fs.Bool("debug", false, "Log every SET")
	followRedirs := fs.Bool("cluster-follow-redirects", true, "ClusterFollowRedirections (cluster only)")
	readReplica := fs.Bool("cluster-read-from-replica", false, "ClusterReadFromReplica (cluster only)")
	// flag.ExitOnError handles parse errors itself (calls os.Exit(2)); no
	// post-Parse error path to handle here.
	_ = fs.Parse(args)

	if *wait > 0 {
		duration = wait
	}
	if *workers < 1 {
		log.Fatalf("workers must be >= 1, got %d", *workers)
	}
	if *rate <= 0 {
		log.Fatalf("rate must be > 0, got %f", *rate)
	}
	// Redis PX requires a positive integer count of milliseconds; sub-ms
	// durations truncate to 0 via .Milliseconds(), and SET ... PX 0 returns
	// "ERR invalid expire time in set" which would silently inflate the
	// errors counter.
	if *keyTTL < time.Millisecond {
		log.Fatalf("key-ttl must be >= 1ms, got %s", *keyTTL)
	}
	// Guard against time.NewTicker(0) panic for very large rates.
	period := time.Duration(float64(time.Second) / *rate)
	if period <= 0 {
		log.Fatalf("rate %.2f/s produces a non-positive ticker period (%s); choose a smaller rate", *rate, period)
	}

	pool, err := redis.NewPool(redis.PoolConfig{
		Server:                    *addr,
		UseTLS:                    false,
		ClusterFollowRedirections: *followRedirs,
		ClusterReadFromReplica:    *readReplica,
		MaxIdleConns:              *workers * 2,
		MaxOpenConns:              *workers * 4,
		ConnTimeout:               5 * time.Second,
		ReadTimeout:               5 * time.Second,
		WriteTimeout:              5 * time.Second,
	})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	log.Printf("write mode: addr=%s workers=%d rate=%.2f/s duration=%s",
		*addr, *workers, *rate, *duration)

	var sets, errs atomic.Int64
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	start := time.Now()

	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ticker := time.NewTicker(period)
			defer ticker.Stop()
			i := *indexStart
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					key := fmt.Sprintf("%sw%d_%d", *keyPrefix, id, i)
					conn := redis.ConfigureDoer(pool, pool.Get())
					_, err := redigo.String(conn.Do("SET", key, "1", "PX", keyTTL.Milliseconds()))
					conn.Close()
					if err != nil {
						errs.Add(1)
						if *debug {
							log.Printf("SET %s err=%v", key, err)
						}
					} else {
						sets.Add(1)
						if *debug {
							log.Printf("SET %s", key)
						}
					}
					i++
				}
			}
		}(w)
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println()
	fmt.Println("================ summary ================")
	fmt.Printf("elapsed:     %s\n", elapsed)
	fmt.Printf("sets:        %d\n", sets.Load())
	fmt.Printf("errors:      %d\n", errs.Load())
	fmt.Printf("ops/sec:     %.1f\n", float64(sets.Load())/elapsed.Seconds())
}
