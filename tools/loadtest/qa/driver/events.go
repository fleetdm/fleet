package main

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// event is one line of the event log, in the scenario catalogue's shape, so the collector can slice server metrics
// by what was running. Times are UTC with milliseconds; took_ms is the client's view, the server's `took` comes
// from Fleet's own logs.
type event struct {
	Run     string         `json:"run"`
	Kind    string         `json:"kind"` // action | start | stop | trip | resume | warn
	ID      string         `json:"id,omitempty"`
	Verb    string         `json:"verb,omitempty"`
	Start   string         `json:"start"`
	End     string         `json:"end,omitempty"`
	Status  int            `json:"status,omitempty"`
	TookMS  int64          `json:"took_ms,omitempty"`
	Verdict string         `json:"verdict,omitempty"` // ok | slow | broken | skipped
	Error   string         `json:"error,omitempty"`
	Detail  map[string]any `json:"detail,omitempty"`
}

const eventTime = "2006-01-02T15:04:05.000Z"

func ts(t time.Time) string { return t.UTC().Format(eventTime) }

type eventLog struct {
	run string
	mu  sync.Mutex
	out []*json.Encoder
}

func newEventLog(run string, ws ...io.Writer) *eventLog {
	l := &eventLog{run: run}
	for _, w := range ws {
		l.out = append(l.out, json.NewEncoder(w))
	}
	return l
}

func (l *eventLog) write(e event) {
	e.Run = l.run
	if e.Start == "" {
		e.Start = ts(time.Now())
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, enc := range l.out {
		_ = enc.Encode(e) // a full disk or a closed pipe shouldn't stop the workload
	}
}
