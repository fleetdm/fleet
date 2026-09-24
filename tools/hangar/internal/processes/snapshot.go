package processes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ReadLogWindow returns a filtered slice of the structured log store. It
// snapshots the rings under lock, then filters lock-free.
func (m *Manager) ReadLogWindow(source string, sinceMS uint64, levels []string, search *string, maxLines *int) LogWindow {
	m.storeMu.Lock()
	channels := make(map[string][]LogEntry, len(m.logStore))
	for ch, r := range m.logStore {
		channels[ch] = r.snapshot()
	}
	m.storeMu.Unlock()
	return filterLogWindow(channels, source, sinceMS, levels, search, maxLines)
}

// LogsDirPath is the directory holding channel logs and snapshots.
func (m *Manager) LogsDirPath() string { return m.logDir }

// SaveLogSnapshot writes a pre-formatted snapshot to <logDir>/snapshots/.
// The frontend supplies a basename; we reject path separators so a hostile
// webview can't escape the snapshots dir.
func (m *Manager) SaveLogSnapshot(filename, contents string) (string, error) {
	if filename == "" || strings.ContainsAny(filename, `/\`) || strings.Contains(filename, "..") {
		return "", errors.New("invalid filename")
	}
	dir := filepath.Join(m.logDir, "snapshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ClearLogChannel clears the in-memory ring(s) and truncates the on-disk
// file(s). "all" clears every channel.
func (m *Manager) ClearLogChannel(channel string) error {
	m.storeMu.Lock()
	if channel == "all" {
		m.logStore = map[string]*ring{}
	} else {
		delete(m.logStore, channel)
	}
	m.storeMu.Unlock()

	// Drop cached writers (so the truncate isn't shadowed by buffered bytes)
	// and decide which files to clear.
	m.writersMu.Lock()
	var channels []string
	if channel == "all" {
		for ch := range m.logWriters {
			channels = append(channels, ch)
		}
	} else {
		channels = []string{channel}
	}
	for _, ch := range channels {
		if cw := m.logWriters[ch]; cw != nil {
			cw.close()
			delete(m.logWriters, ch)
		}
	}
	m.writersMu.Unlock()

	for _, ch := range channels {
		path := logFilePath(m.logDir, ch)
		if _, err := os.Stat(path); err == nil {
			_ = os.WriteFile(path, []byte{}, 0o644)
		}
		rotated := path + ".1"
		if _, err := os.Stat(rotated); err == nil {
			_ = os.Remove(rotated)
		}
	}
	return nil
}
