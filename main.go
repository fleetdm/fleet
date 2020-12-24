package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/fleetdm/orbit/pkg/insecure"
	"github.com/fleetdm/orbit/pkg/osquery"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	serverURL = "localhost:8080"
	certPath  = "/tmp/fleet.pem"
)

func main() {
	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A powered-up, (near) drop-in replacement for osquery"
	defaultRootDir := "/usr/local/orbit"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "root-dir",
			Usage: "Root directory for Orbit state",
			Value: defaultRootDir,
		},
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Disable TLS certificate verification",
		},
		&cli.StringFlag{
			Name:  "fleet-url",
			Usage: "URL (host:port) to Fleet server",
		},
		&cli.StringFlag{
			Name:  "enroll-secret",
			Usage: "Enroll secret for authenticating to Fleet server",
		},
	}
	app.Action = func(c *cli.Context) error {
		err := initialize(c)
		if err != nil {
			return errors.Wrap(err, "initialize")
		}

		var g run.Group
		var options []func(*osquery.Runner) error

		fleetURL := c.String("fleet_url")

		if c.Bool("insecure") {
			proxy, err := insecure.NewTLSProxy(fleetURL)
			if err != nil {
				return errors.Wrap(err, "create TLS proxy")
			}

			g.Add(
				func() error {
					err := proxy.InsecureServeTLS()
					log.Println(err)
					return err
				},
				func(error) {
					if err := proxy.Close(); err != nil {
						log.Printf("error closing proxy: %v", err)
					}
				},
			)

			// Write cert that proxy uses
			err = ioutil.WriteFile(certPath, []byte(insecure.ServerCert), os.ModePerm)
			if err != nil {
				return errors.Wrap(err, "write server cert")
			}

			// Rewrite URL to the proxy URL
			fleetURL = fmt.Sprintf("localhost:%d", proxy.Port)

			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(fleetURL)),
				osquery.WithFlags([]string{"--tls_server_certs", certPath}),
			)
		}

		if enrollSecret := c.String("enroll_secret"); enrollSecret != "" {
			options = append(options,
				osquery.WithEnv([]string{"ENROLL_SECRET="}),
				osquery.WithFlags([]string{"--enroll_secret_env", "ENROLL_SECRET"}),
			)
		}

		if fleetURL != "" {
			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(fleetURL)),
			)
		}

		options = append(options,
			osquery.WithFlags([]string{"--verbose"}),
		)

		// Create an osquery runner with the provided options
		r, _ := osquery.NewRunner(options...)
		g.Add(r.Execute, r.Interrupt)

		// Install a signal handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

		if err := g.Run(); err != nil {
			fmt.Println(err)
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Println("Error:", err)
	}
}

func initialize(c *cli.Context) error {
	fmt.Println(c.String("root-dir"))
	err := os.MkdirAll(c.String("root-dir"), 0o600)
	if err != nil {
		return errors.Wrap(err, "make root directory")
	}

	return nil
}
