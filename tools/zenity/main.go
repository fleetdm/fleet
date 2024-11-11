package main

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
)

func main() {
	output, exitcode, err := execuser.RunWithOutput("zenity --show-entry")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(output), exitcode)
}
