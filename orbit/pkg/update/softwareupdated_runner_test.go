package update

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/oklog/run"
	"github.com/stretchr/testify/require"
)

func TestSoftwareUpdatedRunner(t *testing.T) {
	var callCount int32
	runFn := func() error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	var g run.Group
	// add the "lead" runner, it will return after 1s and cause all runners to stop
	g.Add(func() error { time.Sleep(time.Second); return nil }, func(error) {})

	// add the softwareupdated runner, should run at least 2 times (leave some
	// wiggle room in case it is so slow on CI that it is not scheduled more
	// often).
	r := NewSoftwareUpdatedRunner(SoftwareUpdatedOptions{Interval: 300 * time.Millisecond, runCmdFn: runFn})
	g.Add(r.Execute, r.Interrupt)

	err := g.Run()
	require.NoError(t, err)
	require.GreaterOrEqual(t, atomic.LoadInt32(&callCount), int32(2))
}
