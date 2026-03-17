//go:build !linux

package main

import (
	"errors"

	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
)

// desktopUserServiceManager is a no-op on non-Linux platforms.
// On macOS and Windows, the existing desktopRunner is used instead.
type desktopUserServiceManager struct {
	interruptCh   chan struct{}
	executeDoneCh chan struct{}
}

func newDesktopUserServiceManager(
	_, _, _ string,
	_ bool,
	_ *token.ReadWriter,
	_, _ []byte,
	_ string,
	_ string,
) *desktopUserServiceManager {
	return &desktopUserServiceManager{
		interruptCh:   make(chan struct{}),
		executeDoneCh: make(chan struct{}),
	}
}

func (m *desktopUserServiceManager) Execute() error {
	defer close(m.executeDoneCh)
	return errors.New("desktop user service manager is only supported on Linux")
}

func (m *desktopUserServiceManager) Interrupt(_ error) {
	close(m.interruptCh)
	<-m.executeDoneCh
}
