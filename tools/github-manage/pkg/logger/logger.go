package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

// Init initializes the logger to write to dgm.log file
func Init() error {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create the log file path
	logFilePath := filepath.Join(cwd, "dgm.log")

	// Open or create the log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return err
	}

	// Create loggers with different prefixes
	InfoLogger = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(logFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

// SetOutput sets a custom output writer for all loggers (useful for testing)
func SetOutput(w io.Writer) {
	InfoLogger.SetOutput(w)
	ErrorLogger.SetOutput(w)
	DebugLogger.SetOutput(w)
}

// Info logs an info message
func Info(v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Println(v...)
	}
}

// Infof logs a formatted info message
func Infof(format string, v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Printf(format, v...)
	}
}

// Error logs an error message
func Error(v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Println(v...)
	}
}

// Errorf logs a formatted error message
func Errorf(format string, v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Printf(format, v...)
	}
}

// Debug logs a debug message
func Debug(v ...interface{}) {
	if DebugLogger != nil {
		DebugLogger.Println(v...)
	}
}

// Debugf logs a formatted debug message
func Debugf(format string, v ...interface{}) {
	if DebugLogger != nil {
		DebugLogger.Printf(format, v...)
	}
}
