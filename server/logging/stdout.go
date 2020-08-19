package logging

import (
	"context"
	"encoding/json"
	"fmt"
)

type stdoutLogWriter struct {
}

func NewStdoutLogWriter() (*stdoutLogWriter, error) {
	return &stdoutLogWriter{}, nil
}

func (l *stdoutLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	for _, log := range logs {
		fmt.Printf("%s\n", log)
	}
	return nil
}
