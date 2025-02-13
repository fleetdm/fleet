package migrations

import (
	"log"
	"os"
)

type Logger struct {
	Info *log.Logger
	Warn *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		Info: log.New(os.Stdout, "INFO: ", log.Ltime),
		Warn: log.New(os.Stderr, "WARNING: ", log.Ltime),
	}
}

var logger = NewLogger()
