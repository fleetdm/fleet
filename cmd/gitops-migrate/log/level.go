package log

var Level = LevelInfo

type level uint8

const (
	LevelDebug level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (l level) String() string {
	switch l {
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
