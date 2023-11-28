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
	samplePath := flag.String("sample_path", "", "Path to a file to write the process samples")

	flag.Parse()

	if *pid == 0 {
		log.Fatal("Missing -pid flag")
	}
	if *samplePath == "" {
		log.Fatal("Missing -sample_path flag")
	}

	sampleFile, err := os.Create(*samplePath)
	if err != nil {
		panic(err)
	}
	defer sampleFile.Close()
	done := watcher.Start(int32(*pid), sampleFile, 1*time.Second)
	defer close(done)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
}
