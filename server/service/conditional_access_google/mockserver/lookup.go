package mockserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"
)

// OAuthConfig is Fleet's Web OAuth client, allowlisted by Google for the lookup.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	PartnerID    string

	// AuthURL, TokenURL and APIEndpoint override Google's, for tests.
	AuthURL     string
	TokenURL    string
	APIEndpoint string
}

const lookupScope = "https://www.googleapis.com/auth/cloud-identity.devices.lookup"

func (o *OAuthConfig) oauth2(redirectURL string) *oauth2.Config {
	ep := google.Endpoint
	if o.AuthURL != "" {
		ep.AuthURL = o.AuthURL
	}
	if o.TokenURL != "" {
		ep.TokenURL = o.TokenURL
	}
	return &oauth2.Config{
		ClientID: o.ClientID, ClientSecret: o.ClientSecret, Endpoint: ep, RedirectURL: redirectURL,
		Scopes: []string{"openid", "email", lookupScope},
	}
}

func (o *OAuthConfig) authCodeURL(redirectURL, state string) string {
	return o.oauth2(redirectURL).AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// lookup is the signed-in user and the DeviceUsers Google found.
type lookup struct {
	Names    []string
	Customer string
	Email    string
}

func (s *Server) lookupIOS(ctx context.Context, p *pendingSignIn, code string) (*lookup, error) {
	if s.cfg.OAuth == nil {
		return s.fakeLookup(p.customerID, p.hostUUID, code)
	}
	return s.cfg.OAuth.lookup(ctx, s.cfg.PublicURL+callbackPath, code, func(c *cloudidentity.DevicesDeviceUsersLookupCall) {
		c.Partner(s.cfg.OAuth.PartnerID).IosDeviceId(p.hostUUID)
	})
}

var errRefused = errors.New("invalid_grant: simulated by the mock")

// fakeLookup finds the signed-in user's DeviceUsers on the iOS records whose deviceId is
// the host UUID. With the Google surface on, it reads the records from it and creates
// them when there are none, so the names can be marked; a seeded record without the user
// is a lookup with no names.
func (s *Server) fakeLookup(customerID, hostUUID, code string) (*lookup, error) {
	switch code {
	case CodeRefused:
		return nil, errRefused
	case CodeNotFound:
		return &lookup{Customer: customerID, Email: s.cfg.FakeEmail}, nil
	}
	records := s.cfg.FakeDeviceRecords
	out := &lookup{Customer: customerID, Email: s.cfg.FakeEmail}

	if !s.cfg.Google {
		for i := range records {
			id := fakeDeviceID(hostUUID, i)
			out.Names = append(out.Names, "devices/"+id+"/deviceUsers/"+fakeUserID(id, s.cfg.FakeEmail))
		}
		return out, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	bare := bareCustomerID(customerID)
	var found []*device
	for _, d := range s.devices {
		if d.customerID == bare && d.deviceType == "IOS" && d.deviceID == hostUUID && d.state != DeviceDeleted {
			found = append(found, d)
		}
	}
	if len(found) == 0 {
		for i := range records {
			id := fakeDeviceID(hostUUID, i)
			if existing := s.devices[id]; existing != nil {
				// Deleted in Google: the device checks in again under a new record.
				id = "mock" + randomHex(8)
			}
			found = append(found, s.seedLocked(SeedDevice{
				CustomerID: customerID, DeviceID: hostUUID, DeviceType: "IOS", Model: "iPad", OSVersion: "iPadOS 18.1",
				Emails: []string{s.cfg.FakeEmail},
			}, id))
		}
	}
	for _, d := range found {
		for _, u := range d.users {
			if strings.EqualFold(u.email, s.cfg.FakeEmail) {
				out.Names = append(out.Names, d.userName(u))
			}
		}
	}
	return out, nil
}

// fakeID is a stable opaque ID derived from seed.
func fakeID(prefix, seed string) string {
	h := sha256.Sum256([]byte(prefix + ":" + seed))
	return prefix + hex.EncodeToString(h[:6])
}

// fakeDeviceID is the resource ID of the i-th fake record of an iOS host.
func fakeDeviceID(hostUUID string, i int) string {
	return fakeID("dev", "IOS:"+hostUUID+"/"+strconv.Itoa(i))
}

// fakeUserID is the same with and without the Google surface, so a host keeps its names
// when the surfaces served change.
func fakeUserID(deviceResourceID, email string) string {
	return fakeID("user", deviceResourceID+"/"+email)
}

// lookup exchanges the code with real Google and, when configure is set, runs the
// DeviceUsers lookup for the signed-in user.
func (o *OAuthConfig) lookup(
	ctx context.Context, redirectURL, code string, configure func(*cloudidentity.DevicesDeviceUsersLookupCall),
) (*lookup, error) {
	if code == "" {
		return nil, errors.New("missing code")
	}
	// The exchange and the lookup share this client's timeout.
	ctx = context.WithValue(ctx, oauth2.HTTPClient, fleethttp.NewClient())
	tok, err := o.oauth2(redirectURL).Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange the code: %w", err)
	}
	out := &lookup{Email: idTokenEmail(tok)}
	if configure == nil {
		return out, nil
	}

	opts := []option.ClientOption{option.WithHTTPClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok)))}
	if o.APIEndpoint != "" {
		opts = append(opts, option.WithEndpoint(o.APIEndpoint))
	}
	svc, err := cloudidentity.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create cloud identity service: %w", err)
	}
	call := svc.Devices.DeviceUsers.Lookup("devices/-/deviceUsers").UserId("me")
	configure(call)
	if err := call.Pages(ctx, func(resp *cloudidentity.GoogleAppsCloudidentityDevicesV1LookupSelfDeviceUsersResponse) error {
		out.Names = append(out.Names, resp.Names...)
		if resp.Customer != "" {
			out.Customer = "C" + bareCustomerID(resp.Customer)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("lookup: %w", err)
	}
	return out, nil
}

// idTokenEmail reads the email claim without checking the signature: the token came
// straight from Google's token endpoint.
func idTokenEmail(tok *oauth2.Token) string {
	var claims struct {
		Email string `json:"email"`
	}
	raw, _ := tok.Extra("id_token").(string)
	jwtClaims(raw, &claims)
	return claims.Email
}
