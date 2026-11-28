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
		flStopFile   = flag.String("stop_file", "", "pause while this file (or s3://bucket/key object) exists; a trip creates it")
		flOnTrip     = flag.String("on_trip", "", "shell command to run once per trip (evidence, a notification); gets DRIVER_TRIP_REASON")
		flNotifyEnv  = flag.String("notify_url_env", "", "environment variable holding a webhook URL (Slack incoming-webhook shape) to tell about trips")
		flUsersEnv   = flag.String("users_secret_env", "", "environment variable holding a secret the driver users' passwords are derived from, instead of keeping them in -state_dir")
		flStateDir   = flag.String("state_dir", "", "where the driver keeps its users' passwords (default: ~/.fleet-driver/<fleet host>)")
		flUserPrefix = flag.String("user_prefix", "driver-user-", "email prefix of the driver's own password users")
		flUsers      = flag.Int("users", 20, "number of driver users")
		flInflight   = flag.Int("max_inflight", 8, "most actions running at once")
		flTimeout    = flag.Duration("request_timeout", 2*time.Minute, "timeout of one API request")
		flRunLabel   = flag.String("run", "", "label for this run in every event (default: the start time)")
		flSummary    = flag.String("summary", "", "print a per-action summary of these comma-separated event-log files and exit")
		flSince      = flag.Duration("since", 24*time.Hour, "with -summary: the window, ending now")
		flLT         = flag.String("lt", "", "load-test instance, fleet-<terraform workspace>: its ECS cluster, Aurora cluster and log group carry this name; enables the checks that read AWS")
		flRegion     = flag.String("aws_region", "us-east-2", "AWS region of the load test")
		flLogJSON    = flag.Bool("log_json", false, "log as JSON lines on stderr (for CloudWatch Logs Insights)")
	)
	flag.Parse()
	if *flSummary != "" {
		now := time.Now()
		return summarize(os.Stdout, strings.Split(*flSummary, ","), now.Add(-*flSince), now)
	}

	var handler slog.Handler = slog.NewTextHandler(os.Stderr, nil)
	if *flLogJSON {
		handler = slog.NewJSONHandler(os.Stderr, nil)
	}
	logger := slog.New(handler)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *flScenarios == "" {
		return errors.New("-scenarios is required")
	}
	specs, checkSpecs, other, err := loadSpecs(*flScenarios)
	if err != nil {
		return err
	}
	if len(other) > 0 {
		logger.InfoContext(ctx, "skipping specs of other kinds", "ids", other)
	}
	if *flOnly != "" {
		want := strings.Split(*flOnly, ",")
		specs = slices.DeleteFunc(specs, func(s *spec) bool { return !slices.Contains(want, s.ID) })
		checkSpecs = slices.DeleteFunc(checkSpecs, func(c *checkSpec) bool { return !slices.Contains(want, c.ID) })
	}
	if len(specs) == 0 && len(checkSpecs) == 0 {
		return errors.New("no specs to run")
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

	g := &guard{onTrip: *flOnTrip, api: a, events: events, logger: logger}
	if *flStopFile != "" {
		if g.stop, err = newStopMarker(ctx, *flStopFile, *flRegion); err != nil {
			return err
		}
	}
	if *flUsersEnv != "" {
		if e.usersSecret = os.Getenv(*flUsersEnv); len(e.usersSecret) < 16 {
			return fmt.Errorf("-users_secret_env: $%s must hold at least 16 characters", *flUsersEnv)
		}
	}
	if *flNotifyEnv != "" {
		if g.notifyURL = os.Getenv(*flNotifyEnv); g.notifyURL == "" {
			return fmt.Errorf("-notify_url_env: $%s is empty", *flNotifyEnv)
		}
	}
	ms := &metricSources{guard: g, api: a}
	if *flLT != "" {
		if ms.aws, err = newAWSSources(ctx, *flLT, *flRegion); err != nil {
			return err
		}
	}
	checks, dropped, err := resolveChecks(checkSpecs, ms)
	if err != nil {
		return err
	}
	for _, d := range dropped {
		logger.WarnContext(ctx, "check metric dropped", "why", d)
		events.write(event{Kind: "setup", Verdict: "skipped", Error: d})
	}
	cr := &checkRunner{checks: checks, guard: g, events: events, logger: logger}

	var ready []*spec
	if len(specs) > 0 {
		if err := e.hosts.start(ctx); err != nil {
			return err
		}
		// Setup creates the driver's own objects. A spec whose setup fails is dropped, not the run.
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
	}

	if *flOnce {
		for _, s := range ready {
			runOne(ctx, s, e, events)
		}
		for _, mc := range checks {
			cr.evaluate(ctx, mc, time.Now())
		}
		return nil
	}

	go g.watchHealthz(ctx)
	ids := make([]string, 0, len(ready)+len(checks))
	for _, s := range ready {
		ids = append(ids, s.ID)
	}
	checkIDs := make([]string, 0, len(checks))
	for _, mc := range checks {
		checkIDs = append(checkIDs, mc.checkID+" "+mc.metric)
	}
	events.write(event{Kind: "start", Detail: map[string]any{"specs": ids, "checks": checkIDs, "seed": seed, "profile": *flProfile != ""}})
	logger.InfoContext(ctx, "workload running", "specs", len(ready), "checks", len(checks), "stop_file", *flStopFile)
	done := make(chan struct{})
	go func() { cr.run(ctx); close(done) }()
	if len(ready) > 0 {
		(&workload{specs: ready, env: e, guard: g, events: events, rng: rng, maxInflight: *flInflight}).run(ctx)
	} else {
		<-ctx.Done()
	}
	<-done
	events.write(event{Kind: "stop"})
	return nil
}
