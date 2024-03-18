//go:build darwin || linux
// +build darwin linux

package main

import (
	_ "embed"

	"github.com/rs/zerolog/log"
)

//go:embed icon_dark.png
var iconDark []byte

func blockWaitForStopEvent(channelId string) error {
	log.Debug().Msg("communication channel helpers are not implemented for this platform")
	return nil
}
