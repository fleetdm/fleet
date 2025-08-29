package log

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
		return colorDBG + "DBG" + colorReset
	case LevelInfo:
		return colorINF + "INF" + colorReset
	case LevelWarn:
		return colorWRN + "WRN" + colorReset
	case LevelError:
		return colorERR + "ERR" + colorReset
	case LevelFatal:
		return colorFTL + "FTL" + colorReset
	default:
		return ""
	}
}
