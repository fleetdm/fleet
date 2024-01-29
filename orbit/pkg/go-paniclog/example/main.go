package main

import (
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/go-paniclog"
)

func main() {
	f, err := os.Create("test.log")
	if err != nil {
		fmt.Println("Error creating file:", err)
		os.Exit(1)
	}

	undo, err := paniclog.RedirectStderr(f)
	if err != nil {
		fmt.Println("Error redirecting stderr:", err)
		os.Exit(1)
	}

	f.Close()

	if os.Getenv("UNDO_PANICLOG") != "" {
		// demonstrates undoing the stderr redirect
		undo() //nolint:errcheck
	}

	panic("this should end up in the file instead of the console")
}
