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
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Disable TLS certificate verification",
		},
		&cli.StringFlag{
			Name:  "fleet_url",
			Usage: "URL (host:port) to Fleet server",
		},
		&cli.StringFlag{
			Name:  "enroll_secret",
			Usage: "Enroll secret for authenticating to Fleet server",
		},
	}
	app.Action = func(c *cli.Context) error {
		proxy, err := insecure.NewTLSProxy(serverURL)
		if err != nil {
			return errors.Wrap(err, "create TLS proxy")
		}

		err = ioutil.WriteFile(certPath, []byte(insecure.ServerCert), os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "write server cert")
		}

		var g run.Group

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

		r, _ := osquery.NewRunner(
			osquery.WithFlags(osquery.FleetFlags(fmt.Sprintf("localhost:%d", proxy.Port))),
			osquery.WithFlags([]string{"--tls_server_certs", certPath}),
			osquery.WithFlags([]string{"--verbose"}),
			osquery.WithEnv([]string{"ENROLL_SECRET=fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn"}),
			osquery.WithFlags([]string{"--enroll_secret_env", "ENROLL_SECRET"}),
		)
		g.Add(r.Execute, r.Interrupt)

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

		err = g.Run()
		fmt.Println(err)

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Println("Error:", err)
	}
}
