//go:build darwin || linux
// +build darwin linux

package main

import (
	_ "embed"

	"github.com/rs/zerolog/log"
)

//go:embed icon_light.png
var iconLight []byte

//go:embed icon_dark.png
var iconDark []byte

func getSystemTheme() (theme, error) {
	log.Debug().Msg("get system theme not implemented for this platform")
	return themeUnknown, nil
}

func watchSystemTheme(_ *iconManager) {
	log.Debug().Msg("watch system theme not implemented for this platform")
}

func blockWaitForStopEvent(channelId string) error {
	log.Debug().Msg("communication channel helpers are not implemented for this platform")
	return nil
}
