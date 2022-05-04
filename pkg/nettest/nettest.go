// Package nettest provides functionality to run tests that access the public network.
package nettest

import (
	"os"
	"sync"
	"testing"
)

var m sync.Mutex

func runSerial(t *testing.T) {
	m.Lock()
	t.Logf("network test start: %s", t.Name())

	t.Cleanup(func() {
		t.Logf("network test done: %s", t.Name())
		m.Unlock()
	})
}

// RunSerial makes sure a caller test runs serially (accross packages).
func RunSerial(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
		return
	}

	runSerial(t)
}
