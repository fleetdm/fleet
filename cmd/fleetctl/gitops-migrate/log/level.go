package log

var Level level = LevelInfo

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
		return "DBG"
	case LevelInfo:
		return "INF"
	case LevelWarn:
		return "WRN"
	case LevelError:
		return "ERR"
	case LevelFatal:
		return "FTL"
	default:
		return ""
	}
}
