package certauth

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/file"
)

func loadAuthMsg() (*mdm.Authenticate, error) {
	b, err := ioutil.ReadFile("../../mdm/testdata/Authenticate.2.plist")
	if err != nil {
		return nil, err
	}
	r, err := mdm.DecodeCheckin(b)
	if err != nil {
		return nil, err
	}
	a, ok := r.(*mdm.Authenticate)
	if !ok {
		return nil, errors.New("not an Authenticate message")
	}
	return a, nil
}

func loadTokenMsg() (*mdm.TokenUpdate, error) {
	b, err := ioutil.ReadFile("../../mdm/testdata/TokenUpdate.2.plist")
	if err != nil {
		return nil, err
	}
	r, err := mdm.DecodeCheckin(b)
	if err != nil {
		return nil, err
	}
	a, ok := r.(*mdm.TokenUpdate)
	if !ok {
		return nil, errors.New("not a TokenUpdate message")
	}
	return a, nil
}

func TestNilCertAuth(t *testing.T) {
	auth, err := loadAuthMsg()
	if err != nil {
		t.Fatal(err)
	}
	certAuth := New(nil, nil)
	if certAuth == nil {
		t.Fatal("New returned nil")
	}
	err = certAuth.Authenticate(&mdm.Request{}, auth)
	if err == nil {
		t.Fatal("expected error, nil returned")
	}
	if !errors.Is(err, ErrMissingCert) {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestCertAuth(t *testing.T) {
	_, crt, err := SimpleSelfSignedRSAKeypair("TESTDEVICE", 1)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := file.New("test-db")
	if err != nil {
		t.Fatal(err)
	}
	certAuth := New(&NopService{}, storage)
	if certAuth == nil {
		t.Fatal("New returned nil")
	}
	token, err := loadTokenMsg()
	if err != nil {
		t.Fatal(err)
	}
	// a non-Auth message without first Auth'ing the cert should
	// generate an ErrNoCertAssoc.
	err = certAuth.TokenUpdate(&mdm.Request{Certificate: crt}, token)
	if err == nil {
		t.Fatal("expected err; nil returned")
	}
	if !errors.Is(err, ErrNoCertAssoc) {
		t.Fatalf("wrong error: %v", err)
	}
	// send another one to make sure we're not accidentally allowing
	// retroactive
	err = certAuth.TokenUpdate(&mdm.Request{Certificate: crt}, token)
	if err == nil {
		t.Fatal("expected err; nil returned")
	}
	if !errors.Is(err, ErrNoCertAssoc) {
		t.Fatalf("wrong error: %v", err)
	}
	authMsg, err := loadAuthMsg()
	if err != nil {
		t.Fatal(err)
	}
	// let's actually associate our cert...
	err = certAuth.Authenticate(&mdm.Request{Certificate: crt}, authMsg)
	if err != nil {
		t.Fatal(err)
	}
	// ... and try again.
	err = certAuth.TokenUpdate(&mdm.Request{Certificate: crt}, token)
	if err != nil {
		t.Fatal(err)
	}
	_, crt2, err := SimpleSelfSignedRSAKeypair("TESTDEVICE", 2)
	if err != nil {
		t.Fatal(err)
	}
	// lets try and spoof our UDID using another certificate (bad!)
	err = certAuth.TokenUpdate(&mdm.Request{Certificate: crt2}, token)
	if err == nil {
		t.Fatal("expected err; nil returned")
	}
	if !errors.Is(err, ErrNoCertAssoc) {
		t.Fatalf("wrong error: %v", err)
	}
	os.RemoveAll("test-db")
}

func TestCertAuthRetro(t *testing.T) {
	_, crt, err := SimpleSelfSignedRSAKeypair("TESTDEVICE", 1)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := file.New("test-db")
	if err != nil {
		t.Fatal(err)
	}
	certAuth := New(&NopService{}, storage, WithAllowRetroactive())
	if certAuth == nil {
		t.Fatal("New returned nil")
	}
	token, err := loadTokenMsg()
	if err != nil {
		t.Fatal(err)
	}
	// usually a non-Auth message without first Auth'ing the cert would
	// generate an ErrNoCertAssoc. instead this should allow us to
	// register a cert.
	err = certAuth.TokenUpdate(&mdm.Request{Certificate: crt}, token)
	if err != nil {
		t.Fatal(err)
	}
	// send another one to make sure we're still associated
	err = certAuth.TokenUpdate(&mdm.Request{Certificate: crt}, token)
	if err != nil {
		t.Fatal(err)
	}
	_, crt2, err := SimpleSelfSignedRSAKeypair("TESTDEVICE", 2)
	if err != nil {
		t.Fatal(err)
	}
	// lets try and spoof our UDID using another certificate (bad!) to
	// make sure we were properly setting retroactive association
	err = certAuth.TokenUpdate(&mdm.Request{Certificate: crt2}, token)
	if err == nil {
		t.Fatal("expected err; nil returned")
	}
	if !errors.Is(err, ErrNoCertReuse) {
		t.Fatalf("wrong error: %v", err)
	}
	os.RemoveAll("test-db")
}
