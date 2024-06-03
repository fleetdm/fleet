package fleet_logs

import (
	"context"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

func TablePlugin() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.IntegerColumn("time"),
		table.TextColumn("level"),
		table.TextColumn("message"),
	}

	return table.NewPlugin("fleet_logs", columns, generate)
}

func generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return nil, nil
}

type Message struct {
	Time    int64
	Level   zerolog.Level
	Message []byte
}

type Logger struct {
	writeMutex   sync.Mutex
	defaultLevel zerolog.Level
	logs         []Message
}

func (l *Logger) Write(message []byte) (int, error) {
	time := time.Now().UnixMilli()
	level := l.defaultLevel

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
