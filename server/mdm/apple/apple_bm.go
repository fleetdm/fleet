package apple_mdm

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	abmctx "github.com/fleetdm/fleet/v4/server/contexts/apple_bm"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	depclient "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
)

// SetABMTokenMetadata uses the provided ABM token to fetch the associated
// metadata and use it to update the rest of the abmToken fields (org name,
// apple ID, renew date). It only sets the data on the struct, it does not
// save it in the DB.
func SetABMTokenMetadata(
	ctx context.Context,
	abmToken *fleet.ABMToken,
	depStorage storage.AllDEPStorage,
	ds fleet.Datastore,
	logger *slog.Logger,
	renewal bool,
) error {
	decryptedToken, err := assets.ABMToken(ctx, ds, abmToken.DepName)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting ABM token")
	}

	return SetDecryptedABMTokenMetadata(ctx, abmToken, decryptedToken, depStorage, ds, logger, renewal)
}

// UnsavedABMTokenDepName is the sentinel dep_name used during the initial
// upload of an ABM token before it is saved to the database. It is no longer
// used as the lookup key once the token is saved — dep_name is set to the
// ConsumerKey from the OAuth1 token at that point.
//
// Deprecated: use the ConsumerKey directly as the dep_name for new uploads.
const UnsavedABMTokenOrgName = "new_abm_token" //nolint:gosec

func SetDecryptedABMTokenMetadata(
	ctx context.Context,
	abmToken *fleet.ABMToken,
	decryptedToken *depclient.OAuth1Tokens,
	depStorage storage.AllDEPStorage,
	ds fleet.Datastore,
	logger *slog.Logger,
	renewal bool,
) error {
	depClient := NewDEPClient(depStorage, ds, logger)

	// Use dep_name (ConsumerKey) as the nano_dep_names lookup key.
	// For a new token being uploaded, DepName is pre-set to the ConsumerKey by
	// the caller (UploadABMToken). For tokens migrated from the single-token
	// world, DepName is set to the organization_name (backfilled by migration).
	depName := abmToken.DepName
	if depName == "" {
		// Token with empty dep_name: a legacy migrated token that needs its
		// dep_name set. Use the ConsumerKey as the key.
		depName = decryptedToken.ConsumerKey
		if depName == "" {
			// Fallback for tests/edge cases where ConsumerKey is empty.
			depName = UnsavedABMTokenOrgName
		}
		// Update the token struct so the caller can persist dep_name to the DB.
		abmToken.DepName = depName
	}

	// Always inject the decrypted token into context so RetrieveAuthTokens can
	// find it without a DB lookup. This is required for new token uploads (token
	// not yet saved to DB) and for renewals (new token replaces old one).
	ctx = abmctx.NewContext(ctx, decryptedToken)

	res, err := depClient.AccountDetail(ctx, depName)
	if err != nil {
		var authErr *depclient.AuthError
		if errors.As(err, &authErr) {
			// authentication failure with 401 unauthorized means that the configured
			// Apple BM certificate and/or token are invalid. Fail with a 400 Bad
			// Request.
			msg := err.Error()
			if authErr.StatusCode == http.StatusUnauthorized {
				msg = "The Apple Business Manager certificate or server token is invalid. Restart Fleet with a valid certificate and token. See https://fleetdm.com/learn-more-about/setup-abm for help."
			}
			return ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message:     msg,
				InternalErr: err,
			}, "apple GET /account request failed with authentication error")
		}
		return ctxerr.Wrap(ctx, err, "apple GET /account request failed")
	}

	if res.AdminID == "" {
		// fallback to facilitator ID, as this is the same information but for
		// older versions of the Apple API.
		// https://github.com/fleetdm/fleet/issues/7515#issuecomment-1346579398
		res.AdminID = res.FacilitatorID
	}

	abmToken.OrganizationName = res.OrgName
	abmToken.AppleID = res.AdminID
	abmToken.RenewAt = decryptedToken.AccessTokenExpiry.UTC()
	return nil
}
