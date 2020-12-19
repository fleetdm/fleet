package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/fleetdm/orbit/src/insecure"
	"github.com/fleetdm/orbit/src/osquery"
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

		osqueryPath, err := exec.LookPath("osqueryd")
		if err != nil {
			log.Fatalf("no osquery found: %v", err)
		}

		proxy, err := insecure.NewTLSProxy("localhost:8080")
		if err != nil {
			return errors.Wrap(err, "create TLS proxy")
		}

		err = ioutil.WriteFile(certPath, []byte(insecure.ServerCert), os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "write server cert")
		}

		ctx, cancel := context.WithCancel(context.Background())
		var g run.Group
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

		_, _ = osqueryPath, cancel
		//cmd := exec.CommandContext()
		//
		return nil
	}

	app.Run(os.Args)
}
