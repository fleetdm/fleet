package nanomdm

import (
	"bytes"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
)

type fauxStore struct {
	ua *mdm.UserAuthenticate
}

func (f *fauxStore) StoreUserAuthenticate(_ *mdm.Request, msg *mdm.UserAuthenticate) error {
	f.ua = msg
	return nil
}

func newMDMReq() *mdm.Request {
	return &mdm.Request{EnrollID: &mdm.EnrollID{ID: "<test>"}}
}

func TestUAServiceReject(t *testing.T) {
	store := &fauxStore{}
	s := NewUAService(store, false)
	_, err := s.UserAuthenticate(newMDMReq(), &mdm.UserAuthenticate{})
	var httpErr *service.HTTPStatusError
	if !errors.As(err, &httpErr) {
		// should be returning a HTTPStatusError (to deny management)
		t.Fatalf("no error or incorrect error type")
	}
	if httpErr.Status != 410 {
		// if we've kept the "send-empty" false this needs to return a 410
		// i.e. decline management of the user.
		t.Error("status not 410")
	}
}

func TestUAService(t *testing.T) {
	store := &fauxStore{}
	s := NewUAService(store, true)
	ret, err := s.UserAuthenticate(newMDMReq(), &mdm.UserAuthenticate{})
	if err != nil {
		// should be no error
		t.Fatal(err)
	}
	if !bytes.Equal(ret, emptyDigestChallengeBytes) {
		t.Error("response bytes not equal")
	}
	// second request with DigestResponse
	ret, err = s.UserAuthenticate(newMDMReq(), &mdm.UserAuthenticate{DigestResponse: "test"})
	if err != nil {
		// should be no error
		t.Fatal(err)
	}
	if ret != nil {
		t.Error("response bytes not empty")
	}

}
