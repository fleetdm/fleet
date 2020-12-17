package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/fleetdm/orbit/src/osquery"
	"github.com/oklog/run"
)

func main() {
	fmt.Println("Hello osquery!")

	osqueryPath, err := exec.LookPath("osqueryd")
	if err != nil {
		log.Fatalf("no osquery found: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var g run.Group
	g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

	r, _ := osquery.NewRunner(
		osquery.WithFlags(osquery.FleetFlags("localhost:8080")),
	)
	g.Add(r.Execute, r.Interrupt)

	err = g.Run()
	fmt.Println(err)

	_, _ = osqueryPath, cancel
	//cmd := exec.CommandContext()
}
