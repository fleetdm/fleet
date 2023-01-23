//go:build fakeserial
// +build fakeserial

package osquery

import (
	"fmt"

	"github.com/kolide/kit/ulid"
)

var fakeSerialNumber = fmt.Sprintf("fake%s", ulid.New())

func serialForRow(row map[string]string) string {
	return fakeSerialNumber
}
