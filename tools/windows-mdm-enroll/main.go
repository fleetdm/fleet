package main

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
)

func main() {
	err := update.RunWindowsMDMEnrollment()
	fmt.Println(err)
}
