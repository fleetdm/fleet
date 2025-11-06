package mdm

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// We take in the AndroidDatastore here, so it can also be called from the android package until https://github.com/fleetdm/fleet/issues/31218 is done
func RequiresEnrollOTAAuthentication(ctx context.Context, ds fleet.AndroidDatastore, enrollSecret string, noTeamIdPEnabled bool) (bool, error) {
	secret, err := ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil && !fleet.IsNotFound(err) {
		return false, ctxerr.Wrap(ctx, err, "verify enroll secret")
	}

	if secret == nil {
		// enroll secret is invalid, check if any team has IdP enabled for setup
		// experience and if so require authentication before going through (we
		// enforce the failure due to the enroll secret being invalid only when the
		// enrollment profile is installed).
		ids, err := ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "get team IDs with setup experience IdP enabled")
		}
		return len(ids) > 0, nil
	}

	if secret.TeamID == nil { // enroll in "no team"
		return noTeamIdPEnabled, nil
	}

	tm, err := ds.Team(ctx, *secret.TeamID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "get team for settings")
	}
	return tm.Config.MDM.MacOSSetup.EnableEndUserAuthentication, nil
}
