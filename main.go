package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/fleetdm/orbit/pkg/constant"
	"github.com/fleetdm/orbit/pkg/database"
	"github.com/fleetdm/orbit/pkg/insecure"
	"github.com/fleetdm/orbit/pkg/osquery"
	"github.com/fleetdm/orbit/pkg/update"
	"github.com/fleetdm/orbit/pkg/update/badgerstore"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	serverURL = "localhost:8080"
	notaryURL = "https://tuf.fleetctl.com"
	certPath  = "/tmp/fleet.pem"
)

func main() {
	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A powered-up, (near) drop-in replacement for osquery"
	defaultRootDir := "/usr/local/fleet"
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
			Value: serverURL,
		},
		&cli.StringFlag{
			Name:  "notary-url",
			Usage: "URL (host:port) to Notary update server",
			Value: notaryURL,
		},
		&cli.StringFlag{
			Name:  "enroll-secret",
			Usage: "Enroll secret for authenticating to Fleet server",
		},
	}
	app.Action = func(c *cli.Context) error {
		if err := os.MkdirAll(c.String("root-dir"), constant.DefaultDirMode); err != nil {
			return errors.Wrap(err, "initialize root dir")
		}

		db, err := database.Open(filepath.Join(c.String("root-dir"), "orbit.db"))
		if err != nil {
			return err
		}
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("Error closing badger: %v", err)
			}
		}()

		opt := update.DefaultOptions
		opt.RootDirectory = c.String("root-dir")
		opt.ServerURL = c.String("notary-url")
		opt.LocalStore = badgerstore.New(db.DB)
		updater, err := update.New(opt)
		if err != nil {
			return err
		}
		if err := updater.UpdateMetadata(); err != nil {
			return err
		}
		log.Println(updater.Targets())

		osquerydPath, err := updater.Get("osqueryd", "macos", "stable")
		if err != nil {
			return err
		}

		var g run.Group
		var options []func(*osquery.Runner) error

		fleetURL := c.String("fleet-url")

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

		if enrollSecret := c.String("enroll-secret"); enrollSecret != "" {
			options = append(options,
				osquery.WithEnv([]string{"ENROLL_SECRET=" + enrollSecret}),
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

		options = append(options, osquery.WithPath(osquerydPath))

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
