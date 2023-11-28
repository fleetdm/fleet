package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/watcher"
)

func main() {
	pid := flag.Int("pid", 0, "Process ID")
	name := flag.String("name", "", "Process name")
	samplePath := flag.String("sample_path", "", "Path to a file to write the process samples")

	flag.Parse()

	if *pid == 0 && *name == "" {
		log.Fatal("Missing -pid or -name flag")
	}
	if *pid != 0 && *name != "" {
		log.Fatal("Must specify only one: -pid or -name")
	}
	if *samplePath == "" {
		log.Fatal("Missing -sample_path flag")
	}

	sampleFile, err := os.Create(*samplePath)
	if err != nil {
		panic(err)
	}
	defer sampleFile.Close()
	var done chan struct{}
	if *pid != 0 {
		done = watcher.Start(int32(*pid), sampleFile, 1*time.Second)
	} else {
		done = watcher.StartWithName(*name, sampleFile, 1*time.Second)
	}
	defer close(done)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
}
