package main

// This is a tool to test the zenity package on Linux
// It will show an entry dialog, a progress dialog, and an info dialog

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/zenity"
)

func main() {
	prompt := zenity.New()
	ctx := context.Background()

	output, err := prompt.ShowEntry(ctx, dialog.EntryOptions{
		Title:    "Zenity Test Entry Title",
		Text:     "Zenity Test Entry Text",
		HideText: true,
		TimeOut:  10 * time.Second,
	})
	if err != nil {
		fmt.Println("Err ShowEntry")
		panic(err)
	}

	cancelProgress, err := prompt.ShowProgress(dialog.ProgressOptions{
		Title: "Zenity Test Progress Title",
		Text:  "Zenity Test Progress Text",
	})
	if err != nil {
		fmt.Println("Err ShowProgress")
		panic(err)
	}

	time.Sleep(2 * time.Second)
	if err := cancelProgress(); err != nil {
		fmt.Println("Err cancelProgress")
		panic(err)
	}

	err = prompt.ShowInfo(ctx, dialog.InfoOptions{
		Title:   "Zenity Test Info Title",
		Text:    "Result: " + string(output),
		TimeOut: 10 * time.Second,
	})
	if err != nil {
		fmt.Println("Err ShowInfo")
		panic(err)
	}
}
