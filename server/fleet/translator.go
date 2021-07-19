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

type IDGetter interface {
	GetID() (uint, error)
}

type EmailToUserIDTranslator struct {
	email string
	ds    Datastore
}

func NewEmailToUserIDTranslator(ds Datastore, email string) *EmailToUserIDTranslator {
	return &EmailToUserIDTranslator{email: email, ds: ds}
}

func (e *EmailToUserIDTranslator) GetID() (uint, error) {
	user, err := e.ds.UserByEmail(e.email)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

type LabelNameToIDTranslator struct {
	label string
	ds    Datastore
}

func NewLabelNameToIDTranslator(ds Datastore, label string) *LabelNameToIDTranslator {
	return &LabelNameToIDTranslator{label: label, ds: ds}
}

func (e *LabelNameToIDTranslator) GetID() (uint, error) {
	labelIDs, err := e.ds.LabelIDsByName([]string{e.label})
	if err != nil {
		return 0, err
	}
	return labelIDs[0], nil
}

type TeamNameToIDTranslator struct {
	team string
	ds   Datastore
}

func NewTeamNameToIDTranslator(ds Datastore, team string) *TeamNameToIDTranslator {
	return &TeamNameToIDTranslator{team: team, ds: ds}
}

func (e *TeamNameToIDTranslator) GetID() (uint, error) {
	team, err := e.ds.TeamByName(e.team)
	if err != nil {
		return 0, err
	}
	return team.ID, nil
}

type HostIdentifierToIDTranslator struct {
	host string
	ds   Datastore
}

func NewHostIdentifierToIDTranslator(ds Datastore, host string) *HostIdentifierToIDTranslator {
	return &HostIdentifierToIDTranslator{host: host, ds: ds}
}

func (e *HostIdentifierToIDTranslator) GetID() (uint, error) {
	host, err := e.ds.HostByIdentifier(e.host)
	if err != nil {
		return 0, err
	}
	return host.ID, nil
}

type TranslatorService interface {
	Translate(ctx context.Context, payloads []TranslatePayload) ([]TranslatePayload, error)
}
