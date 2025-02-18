package nanomdm

import (
	"bytes"
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/groob/plist"
)

func newTokenMDMReq() *mdm.Request {
	return &mdm.Request{Context: context.Background()}
}

const tokenTestCheckin = // nolint:gosec // waive G101 hardcoded creds
`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>MessageType</key>
	<string>GetToken</string>
	<key>UDID</key>
	<string>test</string>
	<key>TokenServiceType</key>
	<string>com.apple.maid</string>
</dict>
</plist>
`

func TestTokenFull(t *testing.T) {
	tokenTestData := []byte("hello")

	// create muxer
	m := NewTokenMux()

	// associate a new static token handler with a type
	m.Handle("com.apple.maid", NewStaticToken(tokenTestData))

	// create a new NanoMDM service with our token muxer
	s := New(nil, WithGetToken(m))

	// process GetToken check-in message
	respBytes, err := service.CheckinRequest(s, newTokenMDMReq(), []byte(tokenTestCheckin))
	if err != nil {
		t.Fatal(err)
	}

	// unmarshal response bytes
	resp := new(mdm.GetTokenResponse)
	err = plist.Unmarshal(respBytes, resp)
	if err != nil {
		t.Fatal(err)
	}

	// check that our token data matches
	if want, have := string(tokenTestData), string(resp.TokenData); have != want {
		t.Errorf("have %q; want %q", have, want)
	}
}

func newGetToken(serviceType string, id string) *mdm.GetToken {
	return &mdm.GetToken{
		TokenServiceType: serviceType,
		Enrollment:       mdm.Enrollment{UDID: id},
	}
}

func TestToken(t *testing.T) {
	tokenTestData := []byte("hello")

	// create muxer
	m := NewTokenMux()

	// associate a new static token handler with a type
	m.Handle("com.apple.maid", NewStaticToken(tokenTestData))

	// create a new NanoMDM service with our token muxer
	s := New(nil, WithGetToken(m))

	// dispatch a GetToken check-in message
	resp, err := s.GetToken(newTokenMDMReq(), newGetToken("com.apple.maid", "AAAA-1111"))
	if err != nil {
		t.Fatal(err)
	}

	// check that our token data our matches (from the static handler)
	if !bytes.Equal(tokenTestData, resp.TokenData) {
		t.Error("input and output not equal")
	}

	// supply an invalid service type (not handled) and expect an error
	_, err = s.GetToken(newTokenMDMReq(), newGetToken("com.apple.does-not-exist", "AAAA-1111"))
	if err == nil {
		t.Fatal("should be an error")
	}
}
