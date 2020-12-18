package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/fleetdm/orbit/src/certificate"
	"github.com/fleetdm/orbit/src/osquery"
	"github.com/oklog/run"
	"github.com/urfave/cli/v2"
)

const (
	serverURL = "localhost:8080"
	certPath  = "/tmp/fleet.pem"
)

func main() {
	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A (near) drop-in replacement for osquery with features to ease the deployment to your Fleet."
	app.Action = func(c *cli.Context) error {

		osqueryPath, err := exec.LookPath("osqueryd")
		if err != nil {
			log.Fatalf("no osquery found: %v", err)
		}

		serverCert, err := certificate.FetchPEM(serverURL)
		if err != nil {
			log.Fatalf("retrieve server cert: %v", err)
		}
		err = ioutil.WriteFile(certPath, serverCert, os.ModePerm)
		if err != nil {
			log.Fatalf("write server cert: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		var g run.Group
		g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

		r, _ := osquery.NewRunner(
			osquery.WithFlags(osquery.FleetFlags(serverURL)),
			osquery.WithFlags([]string{"--tls_server_certs", certPath}),
			osquery.WithEnv([]string{"ENROLL_SECRET=fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn"}),
			osquery.WithFlags([]string{"--enroll_secret_env", "ENROLL_SECRET"}),
		)
		g.Add(r.Execute, r.Interrupt)

		err = g.Run()
		fmt.Println(err)

		_, _ = osqueryPath, cancel
		//cmd := exec.CommandContext()
		//
		return nil
	}

	app.Run(os.Args)
}
