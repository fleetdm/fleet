package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
)

type Jira struct {
	ds  fleet.Datastore
	log kitlog.Logger
	// TODO: add jira client
}

func NewJira(ds fleet.Datastore, log kitlog.Logger) *Jira {
	return &Jira{
		ds:  ds,
		log: log,
	}
}

func (j *Jira) Name() string {
	return "jira"
}

func (j *Jira) Run(ctx context.Context, argsJSON json.RawMessage) error {
	// TODO: implement me
	return nil
}
