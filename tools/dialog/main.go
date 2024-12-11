package main

// This is a tool to test the zenity package on Linux
// It will show an entry dialog, a progress dialog, and an info dialog

import (
	"flag"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/kdialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/zenity"
)

func main() {
	dialogTool := flag.String("dialog", "zenity", "Dialog to use: zenity or kdialog")
	flag.Parse()

	var prompt dialog.Dialog

	if *dialogTool == "zenity" {
		fmt.Println("Using zenity")
		prompt = zenity.New()
	} else {
		fmt.Println("Using kdialog")
		prompt = kdialog.New()
	}

	output, err := prompt.ShowEntry(dialog.EntryOptions{
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

	err = prompt.ShowInfo(dialog.InfoOptions{
		Title:   "Zenity Test Info Title",
		Text:    "Result: " + string(output),
		TimeOut: 10 * time.Second,
	})
	if err != nil {
		fmt.Println("Err ShowInfo")
		panic(err)
	}
}
