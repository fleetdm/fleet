// Command verify independently verifies Fleet-maintained app installers
// against their published manifests: it downloads each changed installer,
// recomputes its SHA256 (hash provenance), and checks the publisher's signing
// identity against the pin recorded in the app's input JSON (signer identity
// pinning). See docs/Contributing/research/software/fma-supply-chain-integrity.md.
//
// It runs on the ingest workflow (before the automated update PR opens) and as
// a PR check on ee/maintained-apps/outputs/** changes. Windows Authenticode
// signatures are verified on any OS via osslsigncode; macOS signatures can
// only be verified on a macOS host (otherwise they're deferred to the
// validator, which runs on a real macOS runner).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type config struct {
	repoRoot    string
	changedFrom string
	all         bool
	slugFilter  string
	recordPins  bool
	enforce     bool
	reportPath  string
	reportMD    string
	logger      *slog.Logger
}

func main() {
	cfg := &config{}
	flag.StringVar(&cfg.changedFrom, "changed-from", "origin/main", "git ref to diff ee/maintained-apps/outputs against; only changed installers are verified")
	flag.BoolVar(&cfg.all, "all", false, "verify every app in the catalog instead of only changed ones")
	flag.StringVar(&cfg.slugFilter, "slug", "", "only verify apps whose slug contains this substring")
	flag.BoolVar(&cfg.recordPins, "record-pins", false, "write observed signing identities into input JSONs that have no signature pin yet (backfill)")
	flag.BoolVar(&cfg.enforce, "enforce", false, "exit non-zero when any app fails verification (default is report-only)")
	flag.StringVar(&cfg.reportPath, "report", "verify-report.json", "path to write the JSON verification report")
	flag.StringVar(&cfg.reportMD, "report-md", "", "optional path to write the markdown verification report")
	flag.StringVar(&cfg.repoRoot, "repo-root", ".", "path to the repository root")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	cfg.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	if err := run(context.Background(), cfg); err != nil {
		cfg.logger.Error(fmt.Sprintf("verification failed: %v", err)) //nolint:sloglint // no request context in a CLI entrypoint
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config) error {
	var targets []targetApp
	var err error
	if cfg.all {
		targets, err = allApps(cfg)
	} else {
		targets, err = changedApps(ctx, cfg)
	}
	if err != nil {
		return err
	}

	if cfg.slugFilter != "" {
		filtered := targets[:0]
		for _, t := range targets {
			if strings.Contains(t.Slug, cfg.slugFilter) {
				filtered = append(filtered, t)
			}
		}
		targets = filtered
	}

	mode := "report-only"
	if cfg.enforce {
		mode = "enforce"
	}
	rep := &report{
		BaseRef: cfg.changedFrom,
		Mode:    mode,
		OS:      runtime.GOOS,
	}
	if cfg.all {
		rep.BaseRef = ""
	}

	if len(targets) == 0 {
		cfg.logger.InfoContext(ctx, "no changed maintained apps to verify")
		return emitReport(ctx, cfg, rep)
	}

	tmpDir, err := os.MkdirTemp("", "fma-verify-")
	if err != nil {
		return fmt.Errorf("creating temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			cfg.logger.WarnContext(ctx, fmt.Sprintf("failed to remove temporary directory: %v", err))
		}
	}()

	dl := newDownloader(tmpDir)

	for _, t := range targets {
		cfg.logger.InfoContext(ctx, fmt.Sprintf("Verifying app: %s (%s %s)", t.Name, t.Slug, t.Manifest.Version))
		av := verifyApp(ctx, cfg, dl, t)
		rep.Apps = append(rep.Apps, av)

		// Free disk as we go: a full-catalog run would otherwise accumulate
		// every installer in tmpDir.
		if cfg.all {
			dl.evict(t.Manifest.InstallerURL)
		}
	}

	if cfg.recordPins {
		for _, av := range rep.Apps {
			recorded, err := recordPin(cfg.repoRoot, av)
			if err != nil {
				av.Warnings = append(av.Warnings, fmt.Sprintf("recording pin: %v", err))
				continue
			}
			if recorded {
				cfg.logger.InfoContext(ctx, fmt.Sprintf("Recorded signature pin for %s", av.Slug))
			}
		}
	}

	if err := emitReport(ctx, cfg, rep); err != nil {
		return err
	}

	failures := rep.failures()
	for _, av := range failures {
		cfg.logger.WarnContext(ctx, fmt.Sprintf("App failed verification: %s (%s)", av.Slug, strings.Join(av.Failures, "; ")))
	}
	if cfg.enforce && len(failures) > 0 {
		return fmt.Errorf("%d app(s) failed verification", len(failures))
	}
	if len(failures) > 0 {
		cfg.logger.WarnContext(ctx, fmt.Sprintf("%d app(s) failed verification (report-only mode; not failing)", len(failures)))
	}
	return nil
}

func emitReport(ctx context.Context, cfg *config, rep *report) error {
	jsonBytes, err := rep.json()
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}
	if err := os.WriteFile(cfg.reportPath, jsonBytes, 0o644); err != nil {
		return fmt.Errorf("writing JSON report: %w", err)
	}
	cfg.logger.InfoContext(ctx, fmt.Sprintf("Wrote JSON report to %s", cfg.reportPath))

	md := rep.markdown()
	if cfg.reportMD != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.reportMD), 0o755); err != nil {
			return fmt.Errorf("creating markdown report directory: %w", err)
		}
		if err := os.WriteFile(cfg.reportMD, []byte(md), 0o644); err != nil {
			return fmt.Errorf("writing markdown report: %w", err)
		}
		cfg.logger.InfoContext(ctx, fmt.Sprintf("Wrote markdown report to %s", cfg.reportMD))
	}

	// Surface the report in the GitHub Actions job summary when running in CI.
	if summaryPath := os.Getenv("GITHUB_STEP_SUMMARY"); summaryPath != "" {
		f, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			cfg.logger.WarnContext(ctx, fmt.Sprintf("opening GITHUB_STEP_SUMMARY: %v", err))
			return nil
		}
		defer f.Close()
		if _, err := f.WriteString(md + "\n"); err != nil {
			cfg.logger.WarnContext(ctx, fmt.Sprintf("appending to GITHUB_STEP_SUMMARY: %v", err))
		}
	}
	return nil
}
