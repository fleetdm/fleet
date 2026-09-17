// Command govulndb-mirror mirrors the Go vulnerability database (https://vuln.go.dev) into the
// gzipped JSON artifact that Fleet servers read from the fleetdm/vulnerabilities releases.
//
// Fleet never reaches vuln.go.dev directly: mirroring keeps a single egress point for restricted
// deployments, the same way the OSV feeds are mirrored. A job in fleetdm/vulnerabilities runs
// this command and attaches what it writes to a release. The consumer is
// server/vulnerabilities/govulndb, which also owns the artifact contract this command imports;
// its README.md documents it.
//
// The command does not fail the workflow over upstream trouble. When the collected database
// looks wrong it skips the publish, says on its step outputs which check tripped and by how
// much, and exits zero. Fleet reads a database with fewer advisories as those advisories having
// been remediated and deletes the software_cve rows, so publishing a partial snapshot would
// clear real vulnerabilities from customer hosts. Publishing nothing leaves the last good
// artifact in place and costs only freshness.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/govulndb"
)

const (
	// defaultBaseURL is the Go vulnerability database.
	defaultBaseURL = "https://vuln.go.dev"

	// defaultMaxDropPercent is how much smaller than the previously published artifact a
	// snapshot may be before it is treated as a bad upstream day. The database is append-mostly:
	// reports are withdrawn one at a time, never in batches, so anything past a few percent is
	// far more likely a broken mirror than real remediation.
	defaultMaxDropPercent = 5.0

	// defaultTimeout bounds the whole run. The database is a single ~3 MB download.
	defaultTimeout = 10 * time.Minute
)

type config struct {
	baseURL        string
	outputDir      string
	previous       string
	firstRun       bool
	maxDropPercent float64
	timeout        time.Duration
	now            time.Time
}

func main() {
	cfg, err := parseFlags(os.Args[0], os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	outputs, closeOutputs, err := stepOutputs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	code := execute(ctx, cfg, log.New(os.Stdout, "", log.LstdFlags|log.LUTC), outputs)
	cancel()
	closeOutputs()
	os.Exit(code)
}

// parseFlags reads the command line. Usage text goes to usage.
//
// The drop check is the one thing standing between a half-mirrored database and customer hosts,
// so a missing baseline is an error unless the run says in so many words that it is the first:
// a workflow that quietly stopped passing --previous after a skipped day must not publish blind.
func parseFlags(name string, args []string, usage io.Writer) (config, error) {
	var cfg config
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(usage)
	fs.StringVar(&cfg.outputDir, "output", "", "directory to write the artifact to (required)")
	fs.StringVar(&cfg.previous, "previous", "",
		"path to the previously published artifact, compared against to catch a shrinking database (required unless --first-run)")
	fs.BoolVar(&cfg.firstRun, "first-run", false,
		"no artifact has ever been published, so there is nothing to compare against; skips the drop check")
	fs.StringVar(&cfg.baseURL, "url", defaultBaseURL, "base URL of the Go vulnerability database")
	fs.Float64Var(&cfg.maxDropPercent, "max-drop-percent", defaultMaxDropPercent,
		"how far the module or advisory count may fall below the previously published artifact before the run skips publishing")
	fs.DurationVar(&cfg.timeout, "timeout", defaultTimeout, "overall time limit for the run")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	switch {
	case cfg.outputDir == "":
		return cfg, errors.New("--output is required")
	case cfg.previous == "" && !cfg.firstRun:
		return cfg, errors.New("--previous is required; pass --first-run only when no artifact has ever been published")
	case cfg.previous != "" && cfg.firstRun:
		return cfg, errors.New("--first-run and --previous are mutually exclusive")
	}
	cfg.now = time.Now().UTC().Truncate(time.Second)
	return cfg, nil
}

// execute runs the mirror and returns the process exit code, writing GitHub Actions step outputs
// to outputs. It is where the "never page anyone over a bad upstream day" rule lives: a suspect
// database skips the publish and exits zero, and only a local failure (a broken flag, an
// unwritable output directory) exits non-zero.
func execute(ctx context.Context, cfg config, logger *log.Logger, outputs io.Writer) int {
	path, err := run(ctx, cfg, logger)

	switch {
	case err == nil:
		logger.Printf("published %s", path)
		emit(outputs, "skipped", "false")
		emit(outputs, "artifact", path)
		return 0

	case isSuspect(err):
		logger.Printf("NOT PUBLISHING: %v", err)
		logger.Printf("the latest release keeps no Go vulnerability database asset this run; " +
			"Fleet servers hold on to the last good artifact they downloaded")
		emit(outputs, "skipped", "true")
		emit(outputs, "reason", oneLine(err.Error()))
		return 0

	default:
		logger.Printf("error: %v", err)
		emit(outputs, "skipped", "true")
		emit(outputs, "reason", oneLine(err.Error()))
		return 1
	}
}

// run collects the database, transforms it, checks it against the previously published artifact
// and writes it. It returns the path it wrote, or a suspect error naming the check that tripped.
func run(ctx context.Context, cfg config, logger *log.Logger) (string, error) {
	reports, modified, err := fetchDatabase(ctx, cfg.baseURL, logger)
	if err != nil {
		return "", err
	}
	logger.Printf("collected %d reports from %s (database modified %s)", len(reports), cfg.baseURL, modified)

	artifact, err := transform(reports, cfg.now)
	if err != nil {
		return "", err
	}
	logger.Printf("transformed into %d modules, %d advisories", len(artifact.Modules), countAdvisories(artifact))

	var previous *govulndb.Artifact
	if cfg.firstRun {
		logger.Printf("first run: no previously published artifact to compare against; skipping the drop check")
	} else if previous, err = loadPrevious(cfg.previous); err != nil {
		return "", err
	}

	if err := validate(artifact, previous, cfg.maxDropPercent); err != nil {
		return "", err
	}

	return writeArtifact(artifact, cfg.outputDir, cfg.now)
}

// stepOutputs opens the file GitHub Actions reads step outputs from, or discards them when the
// command runs outside a workflow.
func stepOutputs() (io.Writer, func(), error) {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return io.Discard, func() {}, nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("opening GITHUB_OUTPUT: %w", err)
	}
	return f, func() { f.Close() }, nil
}

func emit(outputs io.Writer, key, value string) {
	fmt.Fprintf(outputs, "%s=%s\n", key, value)
}

// maxReasonLen caps the alert message. Nothing the checks produce comes close, but the reason
// quotes upstream strings (module paths, range types) and an alert payload is not the place to
// find out how long one of those can get.
const maxReasonLen = 500

// oneLine turns a message into a value that survives both GITHUB_OUTPUT's key=value line format
// and interpolation into the alert's JSON payload: one line, no double quotes, no backslashes,
// valid UTF-8. Some of what it carries is upstream text, so it is sanitized rather than trusted.
func oneLine(s string) string {
	s = strings.NewReplacer(`"`, "'", `\`, "").Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxReasonLen {
		cut := maxReasonLen
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + "..."
	}
	return s
}
