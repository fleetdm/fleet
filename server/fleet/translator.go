package fleet

import (
	"context"
	"encoding/json"
)

const (
	TranslatorTypeUserEmail = "User"
	TranslatorTypeLabel     = "Label"
	TranslatorTypeTeam      = "Team"
	TranslatorTypeHost      = "Host"
)

type TranslatePayload struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type StringIdentifierToIDPayload struct {
	Identifier string `json:"identifier"`
	ID         uint   `json:"id"`
}

type TranslatorService interface {
	Translate(ctx context.Context, payloads []TranslatePayload) ([]TranslatePayload, error)
}
