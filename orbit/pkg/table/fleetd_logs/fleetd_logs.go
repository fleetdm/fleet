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
		table.TextColumn("message"),
	}

	return table.NewPlugin("fleetd_logs", columns, generate)
}

func generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	output := []map[string]string{}

	for _, entry := range DefaultLogger.logs {
		row := make(map[string]string, 4)
		row["time"] = entry.Time
		row["level"] = entry.Level.String()
		row["payload"] = string(entry.Payload)
		row["message"] = entry.Message
		output = append(output, row)
	}
	return output, nil
}

type Event struct {
	Time    string
	Level   zerolog.Level
	Payload []byte
	Message string
}

type Logger struct {
	writeMutex sync.Mutex
	logs       []Event
}

func (l *Logger) Write(event []byte) (int, error) {
	msg, err := processLogEntry(event)
	if err != nil {
		return 0, fmt.Errorf("fleet_logs.Write: %w", err)
	}

	if len(msg.Payload) == 0 {
		// If event contains nothing but log level and time but no
		// actual content, return instead of logging it
		return len(event), nil
	}

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, msg)

	if MaxEntries > 0 && len(l.logs) > int(MaxEntries) {
		l.logs = l.logs[len(l.logs)-int(MaxEntries):]
	}

	return len(event), nil
}

func (l *Logger) WriteLevel(level zerolog.Level, event []byte) (int, error) {
	msg, err := processLogEntry(event)
	if err != nil {
		return 0, fmt.Errorf("fleet_logs.WriteLevel: %w", err)
	}

	msg.Level = level

	if len(msg.Payload) == 0 {
		// If event contains nothing but log level and time but no
		// actual content, return instead of logging it
		return len(event), nil
	}

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, msg)

	if MaxEntries > 0 && len(l.logs) > int(MaxEntries) {
		l.logs = l.logs[len(l.logs)-int(MaxEntries):]
	}

	return len(event), nil
}

func processLogEntry(event []byte) (Event, error) {
	var evt map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(event))
	dec.UseNumber()
	if err := dec.Decode(&evt); err != nil {
		return Event{}, fmt.Errorf("cannot decode: %w", err)
	}

	level := zerolog.GlobalLevel()
	var err error
	evtLevel, ok := evt["level"].(string)
	if ok {
		level, err = zerolog.ParseLevel(evtLevel)
		if err != nil {
			return Event{}, fmt.Errorf("unable to parse log event level: %w", err)
		}
		delete(evt, "level")
	}

	var sqliteTime string
	evtTime, ok := evt["time"].(string)
	if ok {
		goTime, err := time.Parse("2006-01-02T15:04:05-07:00", evtTime)
		if err != nil {
			return Event{}, fmt.Errorf("processLogEntry parsing time: %w", err)
		}
		sqliteTime = goTime.UTC().Format(timeFormatString)
		delete(evt, "time")
	} else {
		sqliteTime = time.Now().UTC().Format(timeFormatString)
	}

	evtMessage, ok := evt["message"].(string)
	if ok {
		delete(evt, "message")
	} else {
		evtMessage = ""
	}

	enc, err := json.Marshal(evt)
	if err != nil {
		return Event{}, fmt.Errorf("unable to marshall log event: %w", err)

	}

	msg := Event{
		Time:    sqliteTime,
		Level:   level,
		Payload: enc,
		Message: evtMessage,
	}

	return msg, nil
}
