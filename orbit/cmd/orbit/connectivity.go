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
	Usage: "Verify that the Fleet API endpoints required for enrolled hosts are reachable",
	Description: `Probes the HTTP endpoints documented at
https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet
and reports whether each is reachable from this host's network path.

Authentication is not required; the tool only verifies network reachability.`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "fleet-url",
			Usage:   "URL of the Fleet server to probe (required unless --from-enrollment is set)",
			EnvVars: []string{"ORBIT_FLEET_URL"},
		},
		&cli.StringFlag{
			Name:    "fleet-certificate",
			Usage:   "Path to a custom Fleet server CA certificate bundle",
			EnvVars: []string{"ORBIT_FLEET_CERTIFICATE"},
		},
		&cli.BoolFlag{
			Name:    "insecure",
			Usage:   "Skip TLS certificate verification",
			EnvVars: []string{"ORBIT_INSECURE"},
		},
		&cli.BoolFlag{
			Name:  "from-enrollment",
			Usage: "Read fleet-url and certificate from this host's orbit state in --root-dir",
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
			return connectivity.ListCatalogue(os.Stdout, checks)
		}

		baseURL, rootCAs, insecure, err := resolveTarget(resolveInput{
			fleetURL:       c.String("fleet-url"),
			certPath:       c.String("fleet-certificate"),
			insecure:       c.Bool("insecure"),
			fromEnrollment: c.Bool("from-enrollment"),
			rootDir:        c.String("root-dir"),
		})
		if err != nil {
			return cli.Exit(err.Error(), 2)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		results, err := connectivity.Probe(ctx, connectivity.Options{
			BaseURL:  baseURL,
			RootCAs:  rootCAs,
			Insecure: insecure,
			Timeout:  c.Duration("timeout"),
		}, checks)
		if err != nil {
			return cli.Exit(fmt.Sprintf("probe failed: %s", err), 2)
		}

		if c.Bool("json") {
			if err := connectivity.RenderJSON(os.Stdout, baseURL, results); err != nil {
				return cli.Exit(err.Error(), 2)
			}
		} else {
			if err := connectivity.RenderHuman(os.Stdout, baseURL, results); err != nil {
				return cli.Exit(err.Error(), 2)
			}
		}

		summary := connectivity.Summarize(results)
		switch {
		case summary.Blocked > 0:
			return cli.Exit("", 1)
		case summary.NotFound > 0:
			// Soft warning: endpoints reachable but some routes missing.
			return cli.Exit("", 3)
		}
		return nil
	},
}

type resolveInput struct {
	fleetURL       string
	certPath       string
	insecure       bool
	fromEnrollment bool
	rootDir        string
}

// resolveTarget derives (baseURL, rootCAs, insecure) from the CLI flags,
// including --from-enrollment which reads values from the on-disk orbit state.
func resolveTarget(in resolveInput) (string, *x509.CertPool, bool, error) {
	fleetURL := strings.TrimSpace(in.fleetURL)
	certPath := in.certPath
	insecure := in.insecure

	if in.fromEnrollment {
		rootDir := in.rootDir
		if rootDir == "" {
			return "", nil, false, errors.New("--from-enrollment requires --root-dir (or ORBIT_ROOT_DIR) to be set")
		}

		urlFile := filepath.Join(rootDir, constant.FleetURLFileName)
		b, err := os.ReadFile(urlFile)
		if err != nil {
			return "", nil, false, fmt.Errorf("read fleet url from %s: %w", urlFile, err)
		}
		enrollmentURL := strings.TrimSpace(string(b))
		if enrollmentURL == "" {
			return "", nil, false, fmt.Errorf("fleet url at %s is empty", urlFile)
		}
		if fleetURL != "" && fleetURL != enrollmentURL {
			return "", nil, false, fmt.Errorf("--fleet-url (%s) conflicts with enrollment url (%s); pass only one", fleetURL, enrollmentURL)
		}
		fleetURL = enrollmentURL

		// Fall back to certs.pem in root-dir when --fleet-certificate isn't
		// explicit. This matches how orbit itself resolves the cert.
		if certPath == "" {
			candidate := filepath.Join(rootDir, "certs.pem")
			if _, err := os.Stat(candidate); err == nil {
				certPath = candidate
			}
		}
	}

	if fleetURL == "" {
		return "", nil, false, errors.New("--fleet-url is required (or pass --from-enrollment)")
	}
	if !strings.Contains(fleetURL, "://") {
		fleetURL = "https://" + fleetURL
	}
	if insecure && certPath != "" {
		return "", nil, false, errors.New("--insecure and --fleet-certificate may not be used together")
	}

	var pool *x509.CertPool
	if certPath != "" {
		p, err := certificate.LoadPEM(certPath)
		if err != nil {
			return "", nil, false, fmt.Errorf("load fleet certificate: %w", err)
		}
		pool = p
	}

	return fleetURL, pool, insecure, nil
}
