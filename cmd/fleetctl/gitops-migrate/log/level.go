package log

import "github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/ansi"

var Level level

type level uint8

const (
	LevelDebug level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (self level) String() string {
	switch self {
	case LevelDebug:
		return ansi.BoldBlack + "DBG" + ansi.Reset
	case LevelInfo:
		return ansi.BoldGreen + "INF" + ansi.Reset
	case LevelWarn:
		return ansi.BoldYellow + "WRN" + ansi.Reset
	case LevelError:
		return ansi.BoldRed + "ERR" + ansi.Reset
	case LevelFatal:
		return ansi.BoldRed + "FTL" + ansi.Reset
	default:
		return ""
	}
}
