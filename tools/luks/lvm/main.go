package main

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
)

func main() {
	disk, err := lvm.FindRootDisk()
	if err != nil {
		panic(err)
	}
	fmt.Println("Root Partition:", disk)
}
