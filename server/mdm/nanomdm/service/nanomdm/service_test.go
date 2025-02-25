package nanomdm

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	mock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/micromdm/nanolib/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandAndReportResultsPrimaryDBUse(t *testing.T) {
	ds := new(mock.MDMAppleStore)

	enrollID := &mdm.EnrollID{
		ID:   "1",
		Type: mdm.Device,
	}
	s := Service{
		logger: log.NopLogger,
		store:  ds,
		normalizer: func(e *mdm.Enrollment) *mdm.EnrollID {
			return enrollID
		},
	}
	ds.StoreCommandReportFunc = func(r *mdm.Request, report *mdm.CommandResults) error {
		return nil
	}
	var primaryRequired bool
	ds.RetrieveNextCommandFunc = func(r *mdm.Request, skipNotNow bool) (*mdm.CommandWithSubtype, error) {
		assert.Equal(t, primaryRequired, ctxdb.IsPrimaryRequired(r.Context))
		return nil, nil
	}

	mdmRequest := &mdm.Request{
		Context: context.Background(),
	}
	mdmCommandResults := &mdm.CommandResults{
		Status: "Idle",
	}
	// We don't use primary DB with "Idle" status because we don't update the status of existing commands
	cmd, err := s.CommandAndReportResults(mdmRequest, mdmCommandResults)
	require.NoError(t, err)
	require.Nil(t, cmd)

	// We use primary DB with non-"Idle" status
	mdmCommandResults = &mdm.CommandResults{
		Status: "Acknowledge",
	}
	primaryRequired = true
	cmd, err = s.CommandAndReportResults(mdmRequest, mdmCommandResults)
	require.NoError(t, err)
	require.Nil(t, cmd)

}
