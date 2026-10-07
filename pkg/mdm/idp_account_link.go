package mdm

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type idPAccountGetter interface {
	GetMDMIdPAccountByUUID(ctx context.Context, uuid string) (*fleet.MDMIdPAccount, error)
}

// LogHostIdPAccountLinkChange records a change of the host's IdP account link
// from previousAcctUUID to acctUUID; an empty acctUUID means the link was
// removed. Errors are only logged because the link is already written.
func LogHostIdPAccountLinkChange(
	ctx context.Context,
	ds idPAccountGetter,
	newActivity fleet.NewActivityFunc,
	logger *slog.Logger,
	hostUUID, previousAcctUUID, acctUUID string,
) {
	if previousAcctUUID == acctUUID {
		return
	}

	subjectAcctUUID := acctUUID
	if acctUUID == "" {
		subjectAcctUUID = previousAcctUUID
	}
	email := IdPAccountEmailForActivity(ctx, ds, logger, subjectAcctUUID)
	if email == "" {
		logger.WarnContext(ctx, "skipping idp account link activity: unknown account",
			"host_uuid", hostUUID, "account_uuid", subjectAcctUUID)
		return
	}

	var act fleet.ActivityDetails
	if acctUUID == "" {
		act = fleet.ActivityTypeUnboundHostFromIdPAccount{
			HostUUID: hostUUID,
			IdPEmail: email,
		}
	} else {
		act = fleet.ActivityTypeBoundHostToIdPAccount{
			HostUUID:         hostUUID,
			IdPEmail:         email,
			ReplacedIdPEmail: IdPAccountEmailForActivity(ctx, ds, logger, previousAcctUUID),
		}
	}

	if err := newActivity(ctx, nil, act); err != nil {
		logger.ErrorContext(ctx, "create activity for host idp account link",
			"err", err, "host_uuid", hostUUID, "activity", act.ActivityName())
	}
}

// IdPAccountEmailForActivity resolves an account UUID for a link activity,
// returning an empty string when the account is unknown.
func IdPAccountEmailForActivity(ctx context.Context, ds idPAccountGetter, logger *slog.Logger, acctUUID string) string {
	if acctUUID == "" {
		return ""
	}
	acct, err := ds.GetMDMIdPAccountByUUID(ctx, acctUUID)
	switch {
	case err == nil && acct != nil:
		return acct.Email
	case err != nil && !fleet.IsNotFound(err):
		logger.ErrorContext(ctx, "get idp account for link activity",
			"err", err, "account_uuid", acctUUID)
	}
	return ""
}
