package log

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/inconshreveable/log15"
	"golang.org/x/xerrors"
)

// GetDefaultLogDir returns default log directory
func GetDefaultLogDir() string {
	defaultLogDir := "/var/log/goval-dictionary"
	if runtime.GOOS == "windows" {
		defaultLogDir = filepath.Join(os.Getenv("APPDATA"), "goval-dictionary")
	}
	return defaultLogDir
}

// SetLogger set logger
func SetLogger(logToFile bool, logDir string, debug, logJSON bool) error {
	stderrHandler := log15.StderrHandler
	logFormat := log15.LogfmtFormat()
	if logJSON {
		logFormat = log15.JsonFormatEx(false, true)
		stderrHandler = log15.StreamHandler(os.Stderr, logFormat)
	}

	lvlHandler := log15.LvlFilterHandler(log15.LvlInfo, stderrHandler)
	if debug {
		lvlHandler = log15.LvlFilterHandler(log15.LvlDebug, stderrHandler)
	}

	var handler log15.Handler
	if logToFile {
		if _, err := os.Stat(logDir); err != nil {
			if os.IsNotExist(err) {
				if err := os.Mkdir(logDir, 0700); err != nil {
					return xerrors.Errorf("Failed to create log directory. err: %w", err)
				}
			} else {
				return xerrors.Errorf("Failed to check log directory. err: %w", err)
			}
		}

		logPath := filepath.Join(logDir, "goval-dictionary.log")
		if _, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
			return xerrors.Errorf("Failed to open a log file. err: %w", err)
		}
		handler = log15.MultiHandler(
			log15.Must.FileHandler(logPath, logFormat),
			lvlHandler,
		)
	} else {
		handler = lvlHandler
	}
	log15.Root().SetHandler(handler)
	return nil
}
