package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	httpapi "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/api"
	httpmdm "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/nanomdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/stdlogfmt"
)

const (
	serverURL  = "/mdm"
	enqueueURL = "/api/enq/"
)

// setupNanoMDM configures normal-ish NanoMDM HTTP server handlers for testing.
func setupNanoMDM(logger log.Logger, store storage.AllStorage) (http.Handler, error) {
	// begin with the primary NanoMDM service
	var svc service.CheckinAndCommandService = nanomdm.New(store, nanomdm.WithLogger(logger))

	// chain the certificate auth middleware
	svc = certauth.New(svc, store)

	// setup MDM (check-in and command) handlers
	var mdmHandler http.Handler = httpmdm.CheckinAndCommandHandler(svc, logger.With("handler", "mdm"))
	// mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, , logger.With("handler", "verify"))
	mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, httpmdm.MdmSignatureVerifierFunc(cryptoutil.VerifyMdmSignature))

	// setup API handlers
	var enqueueHandler http.Handler = httpapi.RawCommandEnqueueHandler(store, nil, logger.With("handler", enqueueURL))
	enqueueHandler = http.StripPrefix(enqueueURL, enqueueHandler)

	// create a mux for them
	mux := http.NewServeMux()
	mux.Handle(serverURL, mdmHandler)
	mux.Handle(enqueueURL, enqueueHandler)

	return mux, nil
}

type NanoMDMAPI interface {
	// RawCommandEnqueue enqueues cmd to ids. An APNs push is omitted if nopush is true.
	RawCommandEnqueue(ctx context.Context, ids []string, cmd *mdm.Command, nopush bool) error
}

type IDer interface {
	ID() string
}

func TestE2E(t *testing.T, ctx context.Context, store storage.AllStorage) {
	var logger log.Logger = stdlogfmt.New(stdlogfmt.WithDebugFlag(true))

	mux, err := setupNanoMDM(logger, store)
	if err != nil {
		t.Fatal(err)
	}

	// create a fake HTTP client that dispatches to our raw handlers
	c := NewHandlerClient(mux)

	// create our new device for testing
	d, err := newDeviceFromCheckins(
		c,
		serverURL,
		"../../mdm/testdata/Authenticate.2.plist",
		"../../mdm/testdata/TokenUpdate.2.plist",
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("certauth", func(t *testing.T) { certAuth(t, ctx, store) })
	t.Run("certauth-retro", func(t *testing.T) { certAuthRetro(t, ctx, store) })

	// regression test for retrieving push info of missing devices.
	t.Run("invalid-pushinfo", func(t *testing.T) {
		_, err := store.RetrievePushInfo(ctx, []string{"INVALID"})
		if err != nil {
			// should NOT recieve a "global" error for an enrollment that
			// is merely invalid (or not enrolled yet, or not fully enrolled)
			t.Errorf("should NOT have errored: %v", err)
		}
	})

	t.Run("enroll", func(t *testing.T) { enroll(t, ctx, d, store) })

	t.Run("tally", func(t *testing.T) { tally(t, ctx, d, store, 1) })

	t.Run("bstoken", func(t *testing.T) { bstoken(t, ctx, d.Enrollment) })

	// re-enroll device
	// this is to try and catch any leftover crud that a storage backend didn't
	// clean up (like the tally count, BS token, etc.)
	err = d.DoEnroll(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("re-enrolling device %s: %w", d.ID(), err))
	}

	t.Run("tally-after-reenroll", func(t *testing.T) { tally(t, ctx, d, store, 1) })

	t.Run("bstoken-after-reenroll", func(t *testing.T) { bstoken(t, ctx, d.Enrollment) })

	err = store.ClearQueue(d.NewMDMRequest(ctx))
	if err != nil {
		t.Fatal()
	}

	t.Run("queue", func(t *testing.T) { queue(t, ctx, d, &api{doer: c}) })
}
