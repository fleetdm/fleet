package nanomdm

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// UAService is a basic UserAuthenticate service that optionally implements
// the "zero-length" UserAuthenticate protocol.
// See https://developer.apple.com/documentation/devicemanagement/userauthenticate
type UAService struct {
	logger log.Logger
	store  storage.UserAuthenticateStore

	// By default the UserAuthenticate message will be rejected (410
	// response). If this is set true then a static zero-length
	// digest challenge will be supplied to the first UserAuthenticate
	// check-in message. See the Discussion section of
	// https://developer.apple.com/documentation/devicemanagement/userauthenticate
	sendEmptyDigestChallenge bool
	storeRejectedUserAuth    bool
}

// NewUAService creates a new UserAuthenticate check-in message handler.
func NewUAService(store storage.UserAuthenticateStore, sendEmptyDigestChallenge bool) *UAService {
	return &UAService{
		logger:                   log.NopLogger,
		store:                    store,
		sendEmptyDigestChallenge: sendEmptyDigestChallenge,
	}
}

const emptyDigestChallenge = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>DigestChallenge</key>
	<string></string>
</dict>
</plist>`

var emptyDigestChallengeBytes = []byte(emptyDigestChallenge)

// UserAuthenticate will decline management of a user unless configured
// for the empty digest 2-step UserAuthenticate protocol.
// It implements the NanoMDM service method for UserAuthenticate check-in messages.
func (s *UAService) UserAuthenticate(r *mdm.Request, message *mdm.UserAuthenticate) ([]byte, error) {
	logger := ctxlog.Logger(r.Context, s.logger)
	if s.sendEmptyDigestChallenge || s.storeRejectedUserAuth {
		if err := s.store.StoreUserAuthenticate(r, message); err != nil {
			return nil, err
		}
	}
	// if the DigestResponse is empty then this is the first (of two)
	// UserAuthenticate messages depending on our response
	if message.DigestResponse == "" {
		if s.sendEmptyDigestChallenge {
			logger.Info(
				"msg", "sending empty DigestChallenge response to UserAuthenticate",
			)
			return emptyDigestChallengeBytes, nil
		}
		return nil, service.NewHTTPStatusError(
			http.StatusGone,
			fmt.Errorf("declining management of user: %s", r.ID),
		)
	}
	logger.Debug(
		"msg", "sending empty response to second UserAuthenticate",
	)
	return nil, nil
}
