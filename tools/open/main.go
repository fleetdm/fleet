package main

// This is a tool to test the "open" package.
// It will open the default browser at a given URL.

import (
	"flag"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/open"
)

func main() {
	openTool := flag.String("url", "", "URL to open")
	flag.Parse()

	if *openTool == "" {
		fmt.Println("Please provide a URL to open using the -url flag")
		return
	}

	err := open.Browser(*openTool)
	if err != nil {
		fmt.Println("Err opening URL")
		panic(err)
	}
}
