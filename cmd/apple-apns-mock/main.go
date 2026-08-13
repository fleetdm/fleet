package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	ctx := context.Background()
	listen := flag.String("listen", ":8378", "host:port to listen on")
	sweepInterval := flag.Duration("sweep-interval", 10*time.Minute, "how often to sweep expired pending pushes")
	keepAlive := flag.Duration("keep-alive", 30*time.Second, "how often to send SSE keep-alive pings. Set to 0 to disable.")
	writeTimeout := flag.Duration("write-timeout", 10*time.Second, "deadline for a single SSE write; a device that stops reading is disconnected instead of pinning its token. Set to 0 to disable.")
	defaultTTL := flag.Duration("default-ttl", 24*time.Hour, "how long to hold a push for a disconnected device when the request has no apns-expiration header (an explicit apns-expiration of 0 or a past time means deliver-now-or-discard)")
	debug := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: func() slog.Level {
		if *debug {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	}()}))

	st := newStore(*defaultTTL, logger)
	server := &http.Server{ReadHeaderTimeout: 10 *
		time.Second, WriteTimeout: 0, IdleTimeout: 120 * time.Second, Handler: newMux(st, logger, *keepAlive, *writeTimeout), Addr: *listen}

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

	logger.InfoContext(ctx, "starting mock APNS server", "listen", *listen, "sweep_interval", *sweepInterval, "keep_alive", *keepAlive, "default_ttl", *defaultTTL)
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
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
