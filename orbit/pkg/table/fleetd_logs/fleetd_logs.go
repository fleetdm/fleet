package fleetd_logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

// No timezone, always return in UTC. Use this format because SQLite3
// knows how to parse it.
// See https://www.sqlite.org/lang_datefunc.html
const timeFormatString = "2006-01-02 15:04:05.999999999"

var DefaultLogger = Logger{}
var MaxEntries uint = 10_000

func TablePlugin() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("time"),
		table.TextColumn("level"),
		table.TextColumn("payload"),
	}

	return table.NewPlugin("fleetd_logs", columns, generate)
}

func generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	output := []map[string]string{}

	for _, entry := range DefaultLogger.logs {
		row := make(map[string]string, 3)
		row["time"] = entry.Time
		row["level"] = entry.Level.String()
		row["payload"] = string(entry.Payload)
		output = append(output, row)
	}
	return output, nil
}

type Message struct {
	Time    string
	Level   zerolog.Level
	Payload []byte
}

type Logger struct {
	writeMutex sync.Mutex
	logs       []Message
}

func (l *Logger) Write(message []byte) (int, error) {
	msg, err := processLogEntry(message)
	if err != nil {
		return 0, fmt.Errorf("fleet_logs.Write: %w", err)
	}

	if len(msg.Payload) == 0 {
		// If message contains nothing but log level and time but no
		// actual content, return instead of logging it
		return len(message), nil
	}

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, msg)

	if MaxEntries > 0 && len(l.logs) > int(MaxEntries) {
		l.logs = l.logs[len(l.logs)-int(MaxEntries):]
	}

	return len(message), nil
}

func (l *Logger) WriteLevel(level zerolog.Level, message []byte) (int, error) {
	msg, err := processLogEntry(message)
	if err != nil {
		return 0, fmt.Errorf("fleet_logs.WriteLevel: %w", err)
	}

	msg.Level = level

	if len(msg.Payload) == 0 {
		// If message contains nothing but log level and time but no
		// actual content, return instead of logging it
		return len(message), nil
	}

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, msg)

	if MaxEntries > 0 && len(l.logs) > int(MaxEntries) {
		l.logs = l.logs[len(l.logs)-int(MaxEntries):]
	}

	return len(message), nil
}

func processLogEntry(message []byte) (Message, error) {
	var event map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(message))
	dec.UseNumber()
	if err := dec.Decode(&event); err != nil {
		return Message{}, fmt.Errorf("cannot decode: %w", err)
	}

	level := zerolog.GlobalLevel()
	var err error
	msgLevel, ok := event["level"].(string)
	if ok {
		level, err = zerolog.ParseLevel(msgLevel)
		if err != nil {
			return Message{}, fmt.Errorf("unable to parse log event level: %w", err)
		}
		delete(event, "level")
	}

	var sqliteTime string
	msgTime, ok := event["time"].(string)
	if ok {
		goTime, err := time.Parse("2006-01-02T15:04:05-07:00", msgTime)
		if err != nil {
			return Message{}, fmt.Errorf("processLogEntry parsing time: %w", err)
		}
		sqliteTime = goTime.UTC().Format(timeFormatString)
		delete(event, "time")
	} else {
		sqliteTime = time.Now().UTC().Format(timeFormatString)
	}

	enc, err := json.Marshal(event)
	if err != nil {
		return Message{}, fmt.Errorf("unable to marshall log event: %w", err)

	}

	msg := Message{
		Time:    sqliteTime,
		Level:   level,
		Payload: enc,
	}

	return msg, nil
}
