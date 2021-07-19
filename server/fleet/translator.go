package fleet

import (
	"context"
	"encoding/json"
)

const (
	TranslatorTypeUserEmail = "User"
)

type TranslatePayload struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EmailToIdPayload struct {
	Email string `json:"email"`
	ID    uint   `json:"id"`
}

type TranslatorService interface {
	Translate(ctx context.Context, payloads []TranslatePayload) ([]TranslatePayload, error)
}
