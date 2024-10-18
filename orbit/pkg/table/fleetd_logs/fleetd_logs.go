package fleetd_logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

// No timezone, always return in UTC. Use this format because SQLite3
// knows how to parse it.
// See https://www.sqlite.org/lang_datefunc.html
const timeFormatString = "2006-01-02 15:04:05.999999999"

var Logger = logger{}
var maxEntries = 10_000

func TablePlugin() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("time"),
		table.TextColumn("level"),
		table.TextColumn("payload"),
		table.TextColumn("message"),
		table.TextColumn("error"),
	}

	return table.NewPlugin("fleetd_logs", columns, generate)
}

func generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	output := []map[string]string{}

	for _, entry := range Logger.logs {
		row := make(map[string]string, 5)
		// It would be nice if we could return NULL instead of an
		// empty string when the error is empty
		row["time"] = entry.Time
		row["level"] = entry.Level.String()
		row["payload"] = entry.Payload
		row["message"] = entry.Message
		row["error"] = entry.Error
		output = append(output, row)
	}
	return output, nil
}

type Event struct {
	Time    string
	Level   zerolog.Level
	Payload string
	Message string
	Error   string
}

type logger struct {
	writeMutex sync.Mutex
	logs       []Event
}

func (l *logger) Write(event []byte) (int, error) {
	msgs, err := processLogEntry(event)
	if err != nil {
		return 0, fmt.Errorf("fleet_logs.Write: %w", err)
	}

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, msgs...)

	if maxEntries > 0 && len(l.logs) > maxEntries {
		l.logs = l.logs[len(l.logs)-maxEntries:]
	}

	return len(event), nil
}

func (l *logger) WriteLevel(level zerolog.Level, event []byte) (int, error) {
	msgs, err := processLogEntry(event)
	if err != nil {
		return 0, fmt.Errorf("fleet_logs.WriteLevel: %w", err)
	}

	for idx := range msgs {
		msgs[idx].Level = level
	}

	l.writeMutex.Lock()
	defer l.writeMutex.Unlock()

	l.logs = append(l.logs, msgs...)

	if maxEntries > 0 && len(l.logs) > maxEntries {
		l.logs = l.logs[len(l.logs)-maxEntries:]
	}

	return len(event), nil
}

func processLogEntry(event []byte) ([]Event, error) {
	var evts []map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(event))
	dec.UseNumber()
	for {
		var evt map[string]interface{}
		if err := dec.Decode(&evt); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("cannot decode: %w", err)
		}
		evts = append(evts, evt)
	}

	var entries []Event

	for _, evt := range evts {
		level := zerolog.GlobalLevel()
		var err error
		evtLevel, ok := evt["level"].(string)
		if ok {
			level, err = zerolog.ParseLevel(evtLevel)
			if err != nil {
				return nil, fmt.Errorf("unable to parse log event level: %w", err)
			}
			delete(evt, "level")
		}

		var sqliteTime string
		evtTime, ok := evt["time"].(string)
		if ok {
			goTime, err := time.Parse(time.RFC3339, evtTime)
			if err != nil {
				return nil, fmt.Errorf("processLogEntry parsing time: %w", err)
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

		evtError, ok := evt["error"].(string)
		if ok {
			delete(evt, "error")
		} else {
			evtError = ""
		}

		payload := []byte{}
		if len(evt) > 0 {
			payload, err = json.Marshal(evt)
			if err != nil {
				return nil, fmt.Errorf("unable to marshall log event: %w", err)
			}
		}

		entry := Event{
			Time:    sqliteTime,
			Level:   level,
			Payload: string(payload),
			Message: evtMessage,
			Error:   evtError,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
