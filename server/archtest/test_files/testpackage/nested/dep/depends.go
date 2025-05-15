package dep

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/archtest/test_files/nesteddependency"
)

func init() {
	fmt.Println(nesteddependency.Item)
}
