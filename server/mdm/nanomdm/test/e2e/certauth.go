package e2e

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test/enrollment"
	"github.com/groob/plist"
)

func certAuth(t *testing.T, ctx context.Context, store storage.CertAuthStore) {
	d, auth, tok, err := setupEnrollment()
	if err != nil {
		t.Fatal(err)
	}

	// init service
	svc := certauth.New(&test.NopService{}, store)

	// send a non-Authenticate message (without an initial Authenticate message)
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, certauth.ErrNoCertAssoc)

	// send another one to make sure we're not accidentally allowing retroactive mode
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, certauth.ErrNoCertAssoc)

	// sent an authenticate message. this should associate our cert hash.
	err = svc.Authenticate(d.NewMDMRequest(ctx), auth)
	expectErr(t, err, nil)

	// now send an a message that should be authenticated
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, nil)

	// lets swap out the device identity. i.e. attempt to spoof the device with another cert.
	err = enrollment.ReplaceIdentityRandom(d)
	if err != nil {
		t.Fatal(err)
	}

	// try the spoofed request
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, certauth.ErrNoCertAssoc)
}

func certAuthRetro(t *testing.T, ctx context.Context, store storage.CertAuthStore) {
	d, _, tok, err := setupEnrollment()
	if err != nil {
		t.Fatal(err)
	}

	// init service with retroactive
	svc := certauth.New(&test.NopService{}, store, certauth.WithAllowRetroactive())

	// without retroactive a non-Authenticate message would generate an ErrNoCertAssoc.
	// however with retro on it should allow the association.
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, nil)

	// send another one to make sure the reto association is still good.
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, nil)

	// lets swap out the device identity. i.e. attempt to spoof the device with another cert.
	err = enrollment.ReplaceIdentityRandom(d)
	if err != nil {
		t.Fatal(err)
	}

	// try the spoofed request post-association
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, certauth.ErrNoCertReuse)
}

func expectErr(t *testing.T, have, want error) {
	if !errors.Is(have, want) {
		t.Helper()
		t.Errorf("have: %v; want: %v", have, want)
	}
}

func setupEnrollment() (*enrollment.Enrollment, *mdm.Authenticate, *mdm.TokenUpdate, error) {
	// create our test device
	d, err := enrollment.NewRandomDeviceEnrollment(nil, "com.apple.test-topic", "/", "")
	if err != nil {
		return d, nil, nil, err
	}

	// gen the Authenticate msg and turn into NanoMDM msg
	r, err := d.GenAuthenticate()
	if err != nil {
		return d, nil, nil, err
	}
	auth := new(mdm.Authenticate)
	err = plist.NewDecoder(r).Decode(auth)
	if err != nil {
		return d, auth, nil, err
	}

	// gen the TokenUpdate msg and turn into NanoMDM msg
	r, err = d.GenTokenUpdate()
	if err != nil {
		return d, auth, nil, err
	}
	tok := new(mdm.TokenUpdate)
	err = plist.NewDecoder(r).Decode(tok)
	if err != nil {
		return d, auth, tok, err
	}

	return d, auth, tok, err
}
