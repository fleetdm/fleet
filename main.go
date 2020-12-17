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
)

const (
	serverURL = "localhost:8080"
	certPath  = "/tmp/fleet.pem"
)

func main() {
	fmt.Println("Hello osquery!")

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
	)
	g.Add(r.Execute, r.Interrupt)

	err = g.Run()
	fmt.Println(err)

	_, _ = osqueryPath, cancel
	//cmd := exec.CommandContext()
}
