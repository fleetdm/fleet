package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/connectivity"
	"github.com/urfave/cli/v2"
)

var connectivityCommand = &cli.Command{
	Name:  "connectivity-check",
	Usage: "Verify that this host can reach every Fleet endpoint required for its enrollment",
	Description: `Probes the HTTP endpoints documented at
https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet
against the Fleet server this host is enrolled to, reading the server URL,
certificate, and orbit node key from this host's on-disk orbit state.

Supported endpoints are probed authenticated where possible (with the orbit
node key); the rest are probed unauthenticated and verified by matching the
X-Fleet-Capabilities header or Fleet's JSON error body shape.`,
	Flags: []cli.Flag{
		// Hidden escape hatch for testing and pre-enrollment validation.
		// Disables authenticated probing and skips reading on-disk state.
		&cli.StringFlag{
			Name:    "fleet-url",
			Usage:   "Probe this URL instead of the enrolled server (disables authenticated probes)",
			EnvVars: []string{"ORBIT_FLEET_URL"},
			Hidden:  true,
		},
		&cli.StringFlag{
			Name:    "fleet-certificate",
			Usage:   "Path to a custom Fleet server CA certificate bundle (only used with --fleet-url)",
			EnvVars: []string{"ORBIT_FLEET_CERTIFICATE"},
			Hidden:  true,
		},
		&cli.BoolFlag{
			Name:    "insecure",
			Usage:   "Skip TLS certificate verification (only used with --fleet-url)",
			EnvVars: []string{"ORBIT_INSECURE"},
			Hidden:  true,
		},
		&cli.StringFlag{
			Name:  "features",
			Usage: fmt.Sprintf("Comma-separated subset of features to check (default: all). One or more of: %s", connectivity.FeatureNamesList()),
		},
		&cli.DurationFlag{
			Name:  "timeout",
			Usage: "Per-request timeout",
			Value: 10 * time.Second,
		},
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Emit machine-readable JSON instead of the human-readable report",
		},
		&cli.BoolFlag{
			Name:  "list",
			Usage: "Print the endpoint catalogue without issuing any network requests",
		},
	},
	Action: func(c *cli.Context) error {
		features, err := connectivity.ParseFeatures(c.String("features"))
		if err != nil {
			return cli.Exit(err.Error(), 2)
		}
		checks := connectivity.Catalogue(features...)

		if c.Bool("list") {
			return connectivity.ListCatalogue(c.App.Writer, checks)
		}

		target, err := resolveTarget(resolveInput{
			fleetURLOverride: c.String("fleet-url"),
			certOverride:     c.String("fleet-certificate"),
			insecure:         c.Bool("insecure"),
			rootDir:          c.String("root-dir"),
		})
		if err != nil {
			return cli.Exit(err.Error(), 2)
		}

		timeout := c.Duration("timeout")
		if timeout < 0 {
			return cli.Exit("--timeout must be non-negative", 2)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		results, err := connectivity.Probe(ctx, connectivity.Options{
			BaseURL:      target.baseURL,
			RootCAs:      target.rootCAs,
			Insecure:     target.insecure,
			Timeout:      timeout,
			OrbitNodeKey: target.orbitNodeKey,
		}, checks)
		if err != nil {
			return cli.Exit(fmt.Sprintf("probe failed: %s", err), 2)
		}

		if c.Bool("json") {
			if err := connectivity.RenderJSON(c.App.Writer, target.baseURL, results); err != nil {
				return cli.Exit(err.Error(), 2)
			}
		} else {
			if err := connectivity.RenderHuman(c.App.Writer, target.baseURL, results); err != nil {
				return cli.Exit(err.Error(), 2)
			}
		}

		summary := connectivity.Summarize(results)
		switch {
		case summary.Blocked > 0 || summary.Forbidden > 0 || summary.NotFound > 0:
			return cli.Exit("", 1)
		case summary.NotFleet > 0:
			// Soft warning: endpoint responded but didn't look like Fleet.
			return cli.Exit("", 3)
		}
		return nil
	},
}

type resolveInput struct {
	// fleetURLOverride, when set, bypasses enrollment state lookup and runs
	// in the hidden escape-hatch mode: no on-disk state, no authenticated
	// probes.
	fleetURLOverride string
	certOverride     string
	insecure         bool
	rootDir          string
}

type resolvedTarget struct {
	baseURL      string
	rootCAs      *x509.CertPool
	insecure     bool
	orbitNodeKey string
}

func resolveTarget(in resolveInput) (resolvedTarget, error) {
	if override := strings.TrimSpace(in.fleetURLOverride); override != "" {
		return resolveOverride(override, in.certOverride, in.insecure)
	}
	return resolveFromEnrollment(in.rootDir)
}

// resolveOverride runs in the escape-hatch mode: no enrollment state, no
// orbit node key. Used for pre-enrollment verification and integration tests.
func resolveOverride(fleetURL, certPath string, insecure bool) (resolvedTarget, error) {
	if !strings.Contains(fleetURL, "://") {
		fleetURL = "https://" + fleetURL
	}
	if insecure && certPath != "" {
		return resolvedTarget{}, errors.New("--insecure and --fleet-certificate may not be used together")
	}
	var pool *x509.CertPool
	if certPath != "" {
		p, err := certificate.LoadPEM(certPath)
		if err != nil {
			return resolvedTarget{}, fmt.Errorf("load fleet certificate: %w", err)
		}
		pool = p
	}
	return resolvedTarget{baseURL: fleetURL, rootCAs: pool, insecure: insecure}, nil
}

// resolveFromEnrollment reads the Fleet URL, certificate, and orbit node key
// from the host's on-disk orbit state. This is the default behavior.
func resolveFromEnrollment(rootDir string) (resolvedTarget, error) {
	if rootDir == "" {
		return resolvedTarget{}, errors.New("--root-dir (or ORBIT_ROOT_DIR) must be set to read enrollment state")
	}

	urlFile := filepath.Join(rootDir, constant.FleetURLFileName)
	b, err := os.ReadFile(urlFile)
	if err != nil {
		return resolvedTarget{}, fmt.Errorf("read fleet url from %s (is this host enrolled? pass --fleet-url to probe an unenrolled server): %w", urlFile, err)
	}
	fleetURL := strings.TrimSpace(string(b))
	if fleetURL == "" {
		return resolvedTarget{}, fmt.Errorf("fleet url at %s is empty", urlFile)
	}
	if !strings.Contains(fleetURL, "://") {
		fleetURL = "https://" + fleetURL
	}

	var pool *x509.CertPool
	certPath := filepath.Join(rootDir, "certs.pem")
	switch _, err := os.Stat(certPath); {
	case err == nil:
		p, err := certificate.LoadPEM(certPath)
		if err != nil {
			return resolvedTarget{}, fmt.Errorf("load fleet certificate: %w", err)
		}
		pool = p
	case errors.Is(err, os.ErrNotExist):
		// No custom Fleet cert on disk — fall back to system roots.
	default:
		return resolvedTarget{}, fmt.Errorf("stat fleet certificate %s: %w", certPath, err)
	}

	var orbitNodeKey string
	nodeKeyPath := filepath.Join(rootDir, constant.OrbitNodeKeyFileName)
	keyBytes, err := os.ReadFile(nodeKeyPath)
	switch {
	case err == nil:
		orbitNodeKey = strings.TrimSpace(string(keyBytes))
	case errors.Is(err, os.ErrNotExist):
		// Pre-enrollment or key not yet written — fall back to unauthenticated probing.
	default:
		return resolvedTarget{}, fmt.Errorf("read orbit node key from %s: %w", nodeKeyPath, err)
	}

	return resolvedTarget{baseURL: fleetURL, rootCAs: pool, orbitNodeKey: orbitNodeKey}, nil
}
