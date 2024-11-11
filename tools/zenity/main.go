package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/zenity"
)

func main() {
	prompt := zenity.New()
	ctx := context.Background()

	output, err := prompt.ShowEntry(ctx, zenity.EntryOptions{
		Title:    "Zenity Test Entry Title",
		Text:     "Zenity Test Entry Text",
		HideText: true,
		TimeOut:  10 * time.Second,
	})
	if err != nil {
		fmt.Println("Err ShowEntry")
		panic(err)
	}

	err = prompt.ShowInfo(ctx, zenity.InfoOptions{
		Title:   "Zenity Test Info Title",
		Text:    "Result: " + string(output),
		TimeOut: 10 * time.Second,
	})
	if err != nil {
		fmt.Println("Err ShowInfo")
		panic(err)
	}
}
