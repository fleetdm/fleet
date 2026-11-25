// Command driver runs a load-test instance's admin workload: the action specs in ../scenarios, at the rates of a
// private customer profile, shaped by a business-hours curve. It writes one JSON event per action to stdout. See
// README.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "driver:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		flURL        = flag.String("fleet_url", "", "Fleet server URL")
		flTokenEnv   = flag.String("token_env", "FLEET_DRIVER_TOKEN", "environment variable that holds an admin API token")
		flInsecure   = flag.Bool("insecure", false, "skip TLS certificate verification")
		flScenarios  = flag.String("scenarios", "", "directory of action specs (required)")
		flProfile    = flag.String("profile", "", "private profile file with the measured rates; without it, specs run at their start rates")
		flOnly       = flag.String("only", "", "comma-separated spec IDs to run (default: all)")
		flDryRun     = flag.Duration("dry_run", 0, "print the schedule for this window and exit without calling Fleet, e.g. 24h")
		flOnce       = flag.Bool("once", false, "run each selected spec once, in order, and exit")
		flSeed       = flag.Uint64("seed", 0, "seed for the schedule (default: random)")
		flEvents     = flag.String("events", "", "also append the event log to this file")
		flStopFile   = flag.String("stop_file", "", "pause while this file exists; a trip creates it")
		flOnTrip     = flag.String("on_trip", "", "shell command to run once per trip (evidence, a notification); gets DRIVER_TRIP_REASON")
		flStateDir   = flag.String("state_dir", "", "where the driver keeps its users' passwords (default: ~/.fleet-driver/<fleet host>)")
		flUserPrefix = flag.String("user_prefix", "driver-user-", "email prefix of the driver's own password users")
		flUsers      = flag.Int("users", 20, "number of driver users")
		flInflight   = flag.Int("max_inflight", 8, "most actions running at once")
		flTimeout    = flag.Duration("request_timeout", 2*time.Minute, "timeout of one API request")
		flRunLabel   = flag.String("run", "", "label for this run in every event (default: the start time)")
		flSummary    = flag.String("summary", "", "print a per-action summary of these comma-separated event-log files and exit")
		flSince      = flag.Duration("since", 24*time.Hour, "with -summary: the window, ending now")
	)
	flag.Parse()
	if *flSummary != "" {
		now := time.Now()
		return summarize(os.Stdout, strings.Split(*flSummary, ","), now.Add(-*flSince), now)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *flScenarios == "" {
		return errors.New("-scenarios is required")
	}
	specs, other, err := loadSpecs(*flScenarios)
	if err != nil {
		return err
	}
	if len(other) > 0 {
		logger.InfoContext(ctx, "skipping specs of other kinds", "ids", other)
	}
	if *flOnly != "" {
		want := strings.Split(*flOnly, ",")
		specs = slices.DeleteFunc(specs, func(s *spec) bool { return !slices.Contains(want, s.ID) })
	}
	if len(specs) == 0 {
		return errors.New("no action specs to run")
	}
	prof, err := loadProfile(*flProfile)
	if err != nil {
		return err
	}
	if *flProfile == "" {
		logger.WarnContext(ctx, "no profile: every spec runs at its start rate")
	}
	cs, err := curves(prof.curveConfig())
	if err != nil {
		return err
	}
	if err := resolve(specs, prof, cs); err != nil {
		return err
	}

	seed := *flSeed
	if seed == 0 {
		seed = randSeed()
	}
	rng := newSeeded(seed)
	if *flDryRun > 0 {
		dryRun(os.Stdout, specs, rng, time.Now(), *flDryRun)
		return nil
	}

	if *flURL == "" {
		return errors.New("-fleet_url is required")
	}
	token := os.Getenv(*flTokenEnv)
	if token == "" {
		return fmt.Errorf("no API token in $%s", *flTokenEnv)
	}
	u, err := url.Parse(*flURL)
	if err != nil || u.Host == "" {
		return fmt.Errorf("-fleet_url %q: not a URL", *flURL)
	}
	stateDir := *flStateDir
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		stateDir = filepath.Join(home, ".fleet-driver", u.Host)
	}

	runLabel := *flRunLabel
	if runLabel == "" {
		runLabel = time.Now().UTC().Format("2006-01-02T15:04Z")
	}
	outs := []io.Writer{os.Stdout}
	if *flEvents != "" {
		f, err := os.OpenFile(*flEvents, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // the event log holds no secrets
		if err != nil {
			return err
		}
		defer f.Close()
		outs = append(outs, f)
	}
	events := newEventLog(runLabel, outs...)

	a := newAPI(*flURL, token, *flInsecure, *flTimeout)
	if _, err := a.do(ctx, http.MethodGet, "/api/latest/fleet/me", nil, nil, nil); err != nil {
		return fmt.Errorf("check the token: %w", err)
	}
	e := &env{
		api: a, url: *flURL, insecure: *flInsecure, stateDir: stateDir, logger: logger,
		hosts:      &hostCache{api: a, every: 30 * time.Minute, logger: logger},
		userPrefix: *flUserPrefix, userCount: *flUsers,
	}

	if err := e.hosts.start(ctx); err != nil {
		return err
	}
	// Setup creates the driver's own objects. A spec whose setup fails is dropped, not the run.
	ready := specs[:0]
	for _, s := range specs {
		if err := s.verb.setup(ctx, e); err != nil {
			logger.ErrorContext(ctx, "setup failed, spec dropped", "id", s.ID, "err", err)
			events.write(event{Kind: "setup", ID: s.ID, Verb: s.Verb, Verdict: "broken", Error: err.Error()})
			continue
		}
		ready = append(ready, s)
	}
	if len(ready) == 0 {
		return errors.New("every spec's setup failed")
	}

	if *flOnce {
		for _, s := range ready {
			runOne(ctx, s, e, events)
		}
		return nil
	}

	g := &guard{stopFile: *flStopFile, onTrip: *flOnTrip, api: a, events: events, logger: logger}
	go g.watchHealthz(ctx)
	ids := make([]string, 0, len(ready))
	for _, s := range ready {
		ids = append(ids, s.ID)
	}
	events.write(event{Kind: "start", Detail: map[string]any{"specs": ids, "seed": seed, "profile": *flProfile != ""}})
	logger.InfoContext(ctx, "workload running", "specs", len(ready), "stop_file", *flStopFile)
	(&workload{specs: ready, env: e, guard: g, events: events, rng: rng, maxInflight: *flInflight}).run(ctx)
	events.write(event{Kind: "stop"})
	return nil
}
