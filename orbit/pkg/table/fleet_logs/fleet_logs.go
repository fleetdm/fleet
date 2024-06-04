package fleet_logs

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

var DefaultLogger = Logger{}

func TablePlugin() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.IntegerColumn("time"),
		table.TextColumn("level"),
		table.TextColumn("message"),
	}

	return table.NewPlugin("fleet_logs", columns, generate)
}

func generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	output := make([]map[string]string, len(DefaultLogger.logs))

	for _, entry := range DefaultLogger.logs {
		row := make(map[string]string, 3)
		row["time"] = strconv.FormatInt(entry.Time, 10)
		row["level"] = entry.Level.String()
		row["message"] = string(entry.Message)
		output = append(output, row)
	}
	return output, nil
}

type Message struct {
	Time    int64
	Level   zerolog.Level
	Message []byte
}

type Logger struct {
	writeMutex sync.Mutex
	logs       []Message
}

func (l *Logger) Write(message []byte) (int, error) {
	time := time.Now().UnixMilli()
	level := zerolog.GlobalLevel()

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, Message{
		Time:    time,
		Level:   level,
		Message: message,
	})

	return len(message), nil
}

func (l *Logger) WriteLevel(level zerolog.Level, message []byte) (int, error) {
	time := time.Now().UnixMilli()

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, Message{
		Time:    time,
		Level:   level,
		Message: message,
	})

	return len(message), nil
}
