package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func main() {
	// run owns every defer, so os.Exit never skips cleanup.
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	listen := flag.String("listen", ":8378", "host:port to listen on")
	keepAlive := flag.Duration("keep-alive", 30*time.Second, "how often to send SSE keep-alive pings. Set to 0 to disable.")
	writeTimeout := flag.Duration("write-timeout", 10*time.Second, "deadline for a single SSE write; a device that stops reading is disconnected instead of pinning its token. Set to 0 to disable.")
	defaultTTL := flag.Duration("default-ttl", 24*time.Hour, "how long to hold a push for a disconnected device when the request has no apns-expiration header (an explicit apns-expiration of 0 or a past time means deliver-now-or-discard)")
	debug := flag.Bool("debug", false, "enable debug logging")

	redisAddress := flag.String("redis-address", "", "host:port of the Redis instances share (required)")
	redisUsername := flag.String("redis-username", "", "Redis username")
	redisPassword := flag.String("redis-password", "", "Redis password")
	redisDatabase := flag.Int("redis-database", 0, "Redis database")
	redisUseTLS := flag.Bool("redis-use-tls", false, "connect to Redis over TLS")
	keyPrefix := flag.String("redis-key-prefix", "apns:", "prefix for every Redis key and the push channel; give concurrent load tests different prefixes to isolate them")
	nodeID := flag.String("node-id", "", "identifies this instance in cluster stats (default: hostname-pid)")
	statsInterval := flag.Duration("stats-interval", 5*time.Second, "how often to publish this instance's counters for cluster-wide /stats")

	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Redis is what makes instances interchangeable, so there is no
	// single-instance mode.
	if *redisAddress == "" {
		err := errors.New("-redis-address is required")
		logger.ErrorContext(ctx, err.Error())
		return err
	}
	pool, err := newRedisPool(*redisAddress, *redisUsername, *redisPassword, *redisDatabase, *redisUseTLS)
	if err != nil {
		logger.ErrorContext(ctx, "connect to Redis", "address", *redisAddress, "error", err)
		return err
	}
	defer pool.Close()

	reg := newRegistry()
	coord := newCoordinator(pool, reg, logger, coordinatorConfig{
		NodeID:     resolveNodeID(*nodeID),
		KeyPrefix:  *keyPrefix,
		DefaultTTL: *defaultTTL,
	})
	go coord.Run(ctx, *statsInterval)

	// SSE streams run on hijacked connections, where a keepalive write is the
	// only thing that notices a client that went away (see streamEvents).
	if *keepAlive <= 0 {
		logger.WarnContext(ctx, "keep-alive is disabled: a stream whose client disconnects is not reaped until the next push to that token")
	}

	server := &http.Server{
		Addr:              *listen,
		Handler:           newMux(coord, logger, *keepAlive, *writeTimeout),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0, // SSE streams must not be reaped; writes carry their own deadline
		IdleTimeout:       120 * time.Second,
	}

	logger.InfoContext(ctx, "starting mock APNS server",
		"listen", *listen, "node_id", coord.cfg.NodeID, "redis", *redisAddress,
		"redis_mode", pool.Mode().String(), "key_prefix", *keyPrefix,
		"keep_alive", *keepAlive, "default_ttl", *defaultTTL)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.ErrorContext(ctx, "server stopped", "error", err)
		return err
	}
	return nil
}

// newRedisPool builds the pool the way tools/redis-stress does: no
// config.FleetConfig needed, and NewPool auto-detects cluster mode.
func newRedisPool(address, username, password string, database int, useTLS bool) (fleet.RedisPool, error) {
	return redis.NewPool(redis.PoolConfig{
		Server:                    address,
		Username:                  username,
		Password:                  password,
		Database:                  database,
		UseTLS:                    useTLS,
		ConnTimeout:               5 * time.Second,
		KeepAlive:                 10 * time.Second,
		ConnectRetryAttempts:      3,
		ClusterFollowRedirections: true,
		MaxIdleConns:              8,
		// Every SSE connect claims its pending push, so the pool has to widen
		// at ramp-up rather than serialize.
		MaxOpenConns: 256,
		IdleTimeout:  240 * time.Second,
		// Zero on purpose: the subscribe connection blocks until the next push
		// arrives, and a read deadline would tear it down mid-wait.
		ReadTimeout:  0,
		WriteTimeout: 10 * time.Second,
	})
}

// resolveNodeID labels this instance in cluster stats. Hostname is the ECS
// task ID in the load test; the pid separates instances sharing a host.
func resolveNodeID(configured string) string {
	if configured != "" {
		return configured
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

func newMux(coord *coordinator, logger *slog.Logger, keepAlive, writeTimeout time.Duration) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /memstats", memStatsHandler)

	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		statsHandler(w, r, coord)
	})

	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		eventsSSEHandler(w, r, coord, logger, keepAlive, writeTimeout)
	})

	// Matches APNS HTTP/2 push endpoint for device token hence the weird shape.
	mux.HandleFunc("POST /3/device/{token}", func(w http.ResponseWriter, r *http.Request) {
		pushHandler(w, r, coord, logger)
	})

	return mux
}
