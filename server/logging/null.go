package logging

import (
	"context"
	"encoding/json"
)

type nullLogging struct{}

func (b *nullLogging) Write(ctx context.Context, logs []json.RawMessage) error {
	return nil
}
