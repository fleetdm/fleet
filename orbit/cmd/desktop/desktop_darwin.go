package main

import (
	_ "embed"

	"github.com/rs/zerolog/log"
)

//go:embed fleet-logo.svg
var iconDark []byte

func blockWaitForStopEvent(_ string) error {
	log.Debug().Msg("communication channel helpers are not implemented for this platform")
	return nil
}

func trayIconExists() bool {
	log.Debug().Msg("tray icon checker is not implemented for this platform")
	return true
}

// currentLoginID is only needed for the Windows BitLocker PIN toast.
func currentLoginID() (string, error) {
	return "", nil
}
