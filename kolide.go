package main

//go:generate make generate

import (
	"fmt"
	"math/rand"
	"path"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/kolide/kolide-ose/cli"
	_ "github.com/kolide/kolide-ose/config"
)

// logContextHook is a logrus hook which is used to contextualize application
// logs to include data such as line numbers, file names, etc.
type logContextHook struct{}

// Levels defines which levels the logContextHook logrus hook should apply to
func (hook logContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire defines what the logContextHook should actually do when it is triggered
func (hook logContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(8); ok {
		funcName := runtime.FuncForPC(pc).Name()

		entry.Data["func"] = path.Base(funcName)
		entry.Data["location"] = fmt.Sprintf("%s:%d", path.Base(file), line)
	}

	return nil
}

func init() {
	logrus.AddHook(logContextHook{})
	rand.Seed(time.Now().UnixNano())
}

func main() {
	cli.Launch()
}
