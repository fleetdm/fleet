package fleet

import (
	"context"
)

const (
	TranslatorTypeUserEmail = "user"
	TranslatorTypeLabel     = "label"
	TranslatorTypeTeam      = "team"
	TranslatorTypeHost      = "host"
)

type TranslatePayload struct {
	Type    string                      `json:"type"`
	Payload StringIdentifierToIDPayload `json:"payload"`
}

type StringIdentifierToIDPayload struct {
	Identifier string `json:"identifier"`
	ID         uint   `json:"id"`
}

type TranslatorService interface {
	Translate(ctx context.Context, payloads []TranslatePayload) ([]TranslatePayload, error)
}
