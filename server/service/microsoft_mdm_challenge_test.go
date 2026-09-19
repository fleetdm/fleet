package service

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// A management request for a DeviceID that isn't enrolled must get the same
// challenge response as one for an enrolled device without credentials, so the
// endpoint can't be used to check whether a DeviceID is enrolled.
func TestWindowsManagementUnknownDeviceChallenge(t *testing.T) {
	ds := new(mock.Store)
	kv := new(mock.KVStore)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		KeyValueStore: kv,
	})

	ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
		if mdmDeviceID == "enrolled-device" {
			return &fleet.MDMWindowsEnrolledDevice{
				ID:          1,
				MDMDeviceID: "enrolled-device",
				HostUUID:    "host-uuid-123",
				// Past the initial rekey, so a request without credentials gets a challenge.
				CredentialsAcknowledged: true,
				EnrolledActivityAt:      new(time.Now()),
			}, nil
		}
		return nil, &notFoundError{}
	}
	kvSetCalls := 0
	kv.SetFunc = func(ctx context.Context, key string, value string, expireTime time.Duration) error {
		kvSetCalls++
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "fake-mdm-server.com",
			},
		}, nil
	}

	buildReq := func(deviceID string) *fleet.SyncML {
		payload := `<SyncML xmlns="SYNCML:SYNCML1.2">
  <SyncHdr>
    <VerDTD>1.2</VerDTD>
    <VerProto>DM/1.2</VerProto>
    <SessionID>1</SessionID>
    <MsgID>1</MsgID>
    <Target>
      <LocURI>fake-mdm-server.com</LocURI>
    </Target>
    <Source>
      <LocURI>` + deviceID + `</LocURI>
    </Source>
  </SyncHdr>
  <SyncBody>
    <Alert>
      <CmdID>2</CmdID>
      <Data>1201</Data>
    </Alert>
    <Final />
  </SyncBody>
</SyncML>`
		var req *fleet.SyncML
		require.NoError(t, xml.Unmarshal([]byte(payload), &req))
		return req
	}

	normalize := func(res *fleet.SyncML, deviceID string) string {
		require.NotNil(t, res)
		require.Len(t, res.SyncBody.Raw, 1)
		require.NotNil(t, res.SyncBody.Raw[0].Chal)
		nonce := *res.SyncBody.Raw[0].Chal.Meta.NextNonce.Content
		require.NotEmpty(t, nonce)
		out, err := xml.Marshal(res)
		require.NoError(t, err)
		s := strings.ReplaceAll(string(out), nonce, "NONCE")
		s = strings.ReplaceAll(s, deviceID, "DEVICE")
		// The Status CmdID is a fresh UUID on every response for either kind of
		// caller, so it carries no signal.
		require.NotNil(t, res.SyncBody.Raw[0].CmdID.Value)
		return strings.ReplaceAll(s, res.SyncBody.Raw[0].CmdID.Value, "CMDID")
	}

	resEnrolled, err := svc.GetMDMWindowsManagementResponse(ctx, buildReq("enrolled-device"), nil)
	require.NoError(t, err)
	setCallsAfterEnrolled := kvSetCalls

	resUnknown, err := svc.GetMDMWindowsManagementResponse(ctx, buildReq("unknown-device"), nil)
	require.NoError(t, err)

	require.Equal(t, normalize(resEnrolled, "enrolled-device"), normalize(resUnknown, "unknown-device"))
	// The throwaway nonce for an unknown device is never stored.
	require.Equal(t, setCallsAfterEnrolled, kvSetCalls)
}

// The follow-up probe must be uniform too: a well-formed but wrong credential
// gets the same invalid-credentials response whether the DeviceID is enrolled
// (with or without a stored nonce) or unknown, so a challenge-then-credential
// sequence can't confirm enrollment either.
func TestWindowsManagementCredentialProbeUniform(t *testing.T) {
	ds := new(mock.Store)
	kv := new(mock.KVStore)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		KeyValueStore: kv,
	})

	credsHash := []byte("stored-credentials-hash")
	ds.MDMWindowsGetEnrolledDeviceWithDeviceIDFunc = func(ctx context.Context, mdmDeviceID string) (*fleet.MDMWindowsEnrolledDevice, error) {
		if mdmDeviceID == "enrolled-device" {
			return &fleet.MDMWindowsEnrolledDevice{
				ID:                      1,
				MDMDeviceID:             "enrolled-device",
				HostUUID:                "host-uuid-123",
				CredentialsAcknowledged: true,
				CredentialsHash:         &credsHash,
				EnrolledActivityAt:      new(time.Now()),
			}, nil
		}
		return nil, &notFoundError{}
	}
	kvSetCalls := 0
	kv.SetFunc = func(ctx context.Context, key string, value string, expireTime time.Duration) error {
		kvSetCalls++
		return nil
	}
	storedNonce := ""
	kv.GetFunc = func(ctx context.Context, key string) (*string, error) {
		if storedNonce == "" {
			return nil, nil
		}
		return &storedNonce, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "fake-mdm-server.com",
			},
		}, nil
	}

	// Well-formed MD5 credential with a digest that can't match anything.
	wrongDigest := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	buildCredReq := func(deviceID string) *fleet.SyncML {
		payload := `<SyncML xmlns="SYNCML:SYNCML1.2">
  <SyncHdr>
    <VerDTD>1.2</VerDTD>
    <VerProto>DM/1.2</VerProto>
    <SessionID>1</SessionID>
    <MsgID>1</MsgID>
    <Target>
      <LocURI>fake-mdm-server.com</LocURI>
    </Target>
    <Source>
      <LocURI>` + deviceID + `</LocURI>
    </Source>
    <Cred>
      <Meta>
        <Format xmlns="syncml:metinf">b64</Format>
        <Type xmlns="syncml:metinf">syncml:auth-md5</Type>
      </Meta>
      <Data>` + wrongDigest + `</Data>
    </Cred>
  </SyncHdr>
  <SyncBody>
    <Alert>
      <CmdID>2</CmdID>
      <Data>1201</Data>
    </Alert>
    <Final />
  </SyncBody>
</SyncML>`
		var req *fleet.SyncML
		require.NoError(t, xml.Unmarshal([]byte(payload), &req))
		return req
	}

	normalize := func(res *fleet.SyncML, deviceID string) string {
		require.NotNil(t, res)
		require.Len(t, res.SyncBody.Raw, 1)
		require.NotNil(t, res.SyncBody.Raw[0].Chal)
		nonce := *res.SyncBody.Raw[0].Chal.Meta.NextNonce.Content
		require.NotEmpty(t, nonce)
		out, err := xml.Marshal(res)
		require.NoError(t, err)
		s := strings.ReplaceAll(string(out), nonce, "NONCE")
		s = strings.ReplaceAll(s, deviceID, "DEVICE")
		require.NotEmpty(t, res.SyncBody.Raw[0].CmdID.Value)
		return strings.ReplaceAll(s, res.SyncBody.Raw[0].CmdID.Value, "CMDID")
	}

	// Enrolled device, stored nonce, wrong digest.
	storedNonce = "previously-issued-nonce"
	resEnrolledWithNonce, err := svc.GetMDMWindowsManagementResponse(ctx, buildCredReq("enrolled-device"), nil)
	require.NoError(t, err)

	// Enrolled device, no stored nonce (expired), wrong digest.
	storedNonce = ""
	resEnrolledNoNonce, err := svc.GetMDMWindowsManagementResponse(ctx, buildCredReq("enrolled-device"), nil)
	require.NoError(t, err)

	setCallsBeforeUnknown := kvSetCalls

	// Unknown device, wrong digest.
	resUnknown, err := svc.GetMDMWindowsManagementResponse(ctx, buildCredReq("unknown-device"), nil)
	require.NoError(t, err)

	enrolledWithNonce := normalize(resEnrolledWithNonce, "enrolled-device")
	require.Equal(t, enrolledWithNonce, normalize(resEnrolledNoNonce, "enrolled-device"))
	require.Equal(t, enrolledWithNonce, normalize(resUnknown, "unknown-device"))
	// The unknown device's nonce is a throwaway and never stored.
	require.Equal(t, setCallsBeforeUnknown, kvSetCalls)
}
