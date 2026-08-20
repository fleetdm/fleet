package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	ctx := context.Background()
	listen := flag.String("listen", ":8378", "host:port to listen on")
	sweepInterval := flag.Duration("sweep-interval", 10*time.Minute, "how often to sweep expired pending pushes")
	statsInterval := flag.Duration("stats-interval", 30*time.Second, "how often to log connection, descriptor, thread and memory counts. Set to 0 to disable.")
	keepAlive := flag.Duration("keep-alive", 30*time.Second, "how often to send SSE keep-alive pings. Set to 0 to disable.")
	writeTimeout := flag.Duration("write-timeout", 10*time.Second, "deadline for a single SSE write; a device that stops reading is disconnected instead of pinning its token. Set to 0 to disable.")
	defaultTTL := flag.Duration("default-ttl", 24*time.Hour, "how long to hold a push for a disconnected device when the request has no apns-expiration header (an explicit apns-expiration of 0 or a past time means deliver-now-or-discard)")
	debugLog := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: func() slog.Level {
		if *debugLog {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	}()}))

	st := newStore(*defaultTTL, logger)

	// The runtime's default panic handler writes the stack and exits at once,
	// fast enough that a batching log driver can lose the whole thing: a crash
	// at 140k streams left nothing behind but an exit code. Log it ourselves
	// and hold the process open long enough for the line to ship.
	//
	// This only covers main's own goroutine. Panics in the per-stream
	// goroutines are recovered where they are started (see eventsSSEHandler).
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorContext(ctx, "fatal: unrecovered panic in main",
				"panic", r, "stack", string(debug.Stack()),
				"connected", st.connected.Load(), "os_threads", osThreads())
			flushLogs()
			os.Exit(2)
		}
	}()

	// Being stopped by the orchestrator and dying on our own produced
	// indistinguishable evidence, since neither left a log line. Recording the
	// signal separates them: SIGTERM means something decided to replace us (a
	// deployment, a failed health check), anything else means we broke.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigC
		// Nothing expensive on this path. ECS sends SIGKILL a fixed interval
		// after SIGTERM (30s by default), and counting descriptors takes tens
		// of seconds at 140k streams, so sampling them here would get the
		// process killed before flushLogs could ship the one line that proves
		// the stop was orchestrator-initiated rather than a crash.
		logger.ErrorContext(ctx, "received termination signal, exiting",
			"signal", sig.String(), "connected", st.connected.Load(),
			"os_threads", osThreads())
		flushLogs()
		os.Exit(128 + int(sig.(syscall.Signal)))
	}()

	server := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
		Handler:           newMux(st, logger, *keepAlive, *writeTimeout),
	}

	// SSE streams run on hijacked connections, where a keepalive write is the
	// only thing that notices a client that went away (see streamEvents).
	if *keepAlive <= 0 {
		logger.WarnContext(ctx, "keep-alive is disabled: a stream whose client disconnects is not reaped until the next push to that token")
	}

	if *sweepInterval <= 0 {
		logger.ErrorContext(ctx, "sweep interval cannot be disabled, using default 10m")
		*sweepInterval = 10 * time.Minute
	}
	// Start sweep goroutine to drop expired pending pushes and delete empty entries.
	go func() {
		ticker := time.NewTicker(*sweepInterval)
		defer ticker.Stop()
		for range ticker.C {
			logger.InfoContext(ctx, "sweeping expired pending pushes")
			st.sweep(time.Now())
		}
	}()

	if *statsInterval > 0 {
		go func() {
			ticker := time.NewTicker(*statsInterval)
			defer ticker.Stop()
			for range ticker.C {
				logStats(ctx, logger, st)
			}
		}()
	} else {
		logger.WarnContext(ctx, "stats logging is disabled: a task that dies will leave no record of what it was holding")
	}

	lst, err := net.Listen("tcp", *listen)
	if err != nil {
		logger.ErrorContext(ctx, "failed to listen", "listen", *listen, "error", err)
		flushLogs()
		os.Exit(1)
	}

	logger.InfoContext(ctx, "starting mock APNS server", "listen", *listen, "sweep_interval", *sweepInterval,
		"stats_interval", *statsInterval, "keep_alive", *keepAlive, "default_ttl", *defaultTTL,
		"fd_limit", fdLimit(), "gomaxprocs", runtime.GOMAXPROCS(0))

	// Serve rather than ListenAndServe, so accept failures go through
	// resilientListener instead of ending the process (see listener.go).
	err = server.Serve(&resilientListener{Listener: lst, logger: logger, errs: &st.acceptErrors})
	logger.ErrorContext(ctx, "server stopped serving", "error", err,
		"connected", st.connected.Load(), "accept_errors", st.acceptErrors.Load(),
		"os_threads", osThreads())
	flushLogs()
	os.Exit(1)
}

// flushLogs gives a batching log driver time to ship what was just written.
// Fargate's awslogs driver collects container output in batches, so a process
// that exits immediately after logging its own cause of death can lose exactly
// the line that explains it.
func flushLogs() {
	time.Sleep(2 * time.Second)
}

// logStats emits one line carrying everything needed to explain a death after
// the fact. The server used to log only a 10-minute sweep with no numbers in
// it, so a task killed while holding 140k streams left no record of what it
// was doing.
//
// os_threads earns its place: it can hit a hard ceiling while CPU and memory
// stay flat, it never appears in container metrics, and passing the runtime's
// 10,000-thread limit is a fatal error rather than a panic. It is also cheap,
// being one small read of /proc/self/status.
//
// Everything here has to stay cheap. openFDs is deliberately absent: walking
// /proc/self/fd is O(descriptors) and contends with the file table that every
// accept and close needs, which at 140k streams under an active ramp took
// 30-60 seconds. On a 30s ticker that ran essentially continuously, starving
// the accept path badly enough to fail a 6s health check -- the telemetry
// causing the outage it was meant to explain. fd_limit is kept because it is a
// single getrlimit and it answers the only question that mattered, whether the
// platform actually granted the limit the task definition asked for. Ask for a
// live count explicitly with GET /stats?fds=1, and expect it to be slow.
func logStats(ctx context.Context, logger *slog.Logger, st *store) {
	heap, stacks, inUse := runtimeMem()
	logger.InfoContext(ctx, "stats",
		"connected", st.connected.Load(),
		"pushes_received", st.pushesReceived.Load(),
		"delivered_live", st.deliveredLive.Load(),
		"delivered_on_connect", st.deliveredOnConnect.Load(),
		"stored", st.stored.Load(),
		"coalesced", st.coalesced.Load(),
		"expired", st.expired.Load(),
		"accept_errors", st.acceptErrors.Load(),
		"goroutines", runtime.NumGoroutine(),
		"os_threads", osThreads(),
		"fd_limit", fdLimit(),
		"heap_bytes", heap,
		"stack_bytes", stacks,
		"in_use_bytes", inUse,
	)
}

func newMux(st *store, logger *slog.Logger, keepAlive, writeTimeout time.Duration) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		healthzHandler(w, r)
	})

	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		statsHandler(w, r, st)
	})

	mux.HandleFunc("GET /memstats", func(w http.ResponseWriter, r *http.Request) {
		memStatsHandler(w, r)
	})

	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		eventsSSEHandler(w, r, st, logger, keepAlive, writeTimeout)
	})

	// Matches APNS HTTP/2 push endpoint for device token hence the weird shape.
	mux.HandleFunc("POST /3/device/{token}", func(w http.ResponseWriter, r *http.Request) {
		pushHandler(w, r, st, logger)
	})

	return mux
}
