package contract

import "github.com/fleetdm/fleet/v4/server/fleet"

type ScimDetailsResponse struct {
	fleet.ScimDetails
	Err error `json:"-"`
}

func (r ScimDetailsResponse) Error() error {
	return r.Err
}
