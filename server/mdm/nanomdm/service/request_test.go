package service

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/stretchr/testify/require"
)

// recordingService records which methods are called.
type recordingService struct {
	called []string
}

func (s *recordingService) Authenticate(*mdm.Request, *mdm.Authenticate) error {
	s.called = append(s.called, "Authenticate")
	return nil
}

func (s *recordingService) TokenUpdate(*mdm.Request, *mdm.TokenUpdate) error {
	s.called = append(s.called, "TokenUpdate")
	return nil
}

func (s *recordingService) CheckOut(*mdm.Request, *mdm.CheckOut) error {
	s.called = append(s.called, "CheckOut")
	return nil
}

func (s *recordingService) SetBootstrapToken(*mdm.Request, *mdm.SetBootstrapToken) error {
	s.called = append(s.called, "SetBootstrapToken")
	return nil
}

func (s *recordingService) GetBootstrapToken(*mdm.Request, *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	s.called = append(s.called, "GetBootstrapToken")
	return nil, nil
}

func (s *recordingService) UserAuthenticate(*mdm.Request, *mdm.UserAuthenticate) ([]byte, error) {
	s.called = append(s.called, "UserAuthenticate")
	return nil, nil
}

func (s *recordingService) DeclarativeManagement(*mdm.Request, *mdm.DeclarativeManagement) ([]byte, error) {
	s.called = append(s.called, "DeclarativeManagement")
	return nil, nil
}

func (s *recordingService) GetToken(*mdm.Request, *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	s.called = append(s.called, "GetToken")
	return nil, nil
}

func (s *recordingService) CommandAndReportResults(*mdm.Request, *mdm.CommandResults) (*mdm.Command, error) {
	s.called = append(s.called, "CommandAndReportResults")
	return nil, nil
}

func plistDict(kv ...string) []byte {
	body := ""
	for i := 0; i+1 < len(kv); i += 2 {
		body += fmt.Sprintf("<key>%s</key><string>%s</string>", kv[i], kv[i+1])
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>` + body + `</dict></plist>`)
}

func requireBadRequest(t *testing.T, err error) {
	t.Helper()
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("want an HTTPStatusError, got %v", err)
	}
	require.Equal(t, http.StatusBadRequest, statusErr.Status)
}

func TestCheckinRequestDeviceChannelID(t *testing.T) {
	const victimUserChannel = "C8D90CC1-0C2A-52D4-A6F4-DF55522A740F:5E03D4D6-EDF2-444D-B7E6-1DC2BA3EC334"

	// Every check-in message type is refused before the service sees it.
	for _, messageType := range []string{
		"Authenticate", "TokenUpdate", "CheckOut", "SetBootstrapToken", "GetBootstrapToken",
		"UserAuthenticate", "DeclarativeManagement", "GetToken",
	} {
		t.Run(messageType+" refused", func(t *testing.T) {
			svc := &recordingService{}
			_, err := CheckinRequest(svc, &mdm.Request{}, plistDict("MessageType", messageType, "UDID", victimUserChannel))
			requireBadRequest(t, err)
			require.Empty(t, svc.called)
		})
	}

	t.Run("account-driven enrollment ID refused", func(t *testing.T) {
		svc := &recordingService{}
		_, err := CheckinRequest(svc, &mdm.Request{}, plistDict("MessageType", "Authenticate", "EnrollmentID", "A:B"))
		requireBadRequest(t, err)
		require.Empty(t, svc.called)
	})

	for _, tc := range []struct {
		name string
		kv   []string
	}{
		{"Mac", []string{"UDID", "C8D90CC1-0C2A-52D4-A6F4-DF55522A740F"}},
		{"iPhone", []string{"UDID", "00008150-00191C8E26E0C01C"}},
		{"older iPhone", []string{"UDID", "2b6f0cc904d137be2e1730235f5664094b831186"}},
		{"account-driven", []string{"EnrollmentID", "815AB37A-D8E8-42D1-915F-931BBF05CFC0"}},
		{"user channel", []string{"UDID", "C8D90CC1-0C2A-52D4-A6F4-DF55522A740F", "UserID", "5E03D4D6-EDF2-444D-B7E6-1DC2BA3EC334"}},
	} {
		t.Run(tc.name+" accepted", func(t *testing.T) {
			svc := &recordingService{}
			_, err := CheckinRequest(svc, &mdm.Request{}, plistDict(append([]string{"MessageType", "TokenUpdate"}, tc.kv...)...))
			require.NoError(t, err)
			require.Equal(t, []string{"TokenUpdate"}, svc.called)
		})
	}
}

func TestCommandAndReportResultsRequestDeviceChannelID(t *testing.T) {
	svc := &recordingService{}
	for _, idKey := range []string{"UDID", "EnrollmentID"} {
		_, err := CommandAndReportResultsRequest(svc, &mdm.Request{}, plistDict(idKey, "A:B", "Status", "Idle"))
		requireBadRequest(t, err)
		require.Empty(t, svc.called)
	}

	_, err := CommandAndReportResultsRequest(svc, &mdm.Request{}, plistDict("UDID", "C8D90CC1-0C2A-52D4-A6F4-DF55522A740F", "Status", "Idle"))
	require.NoError(t, err)
	require.Equal(t, []string{"CommandAndReportResults"}, svc.called)
}
