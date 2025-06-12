package testpackage

import (
	"crypto" // for test
	"fmt"

	"github.com/fleetdm/fleet/v4/server/archtest/test_files/dependency"
)

func What(_ crypto.Decrypter) {
	fmt.Println(dependency.Item)
}
