package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"

	"github.com/groob/plist"
)

type HTTPStatusError struct {
	Status int
	Err    error
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP status %d (%s): %v", e.Status, http.StatusText(e.Status), e.Err)
}

func (e *HTTPStatusError) Unwrap() error {
	return e.Err
}

func NewHTTPStatusError(status int, err error) *HTTPStatusError {
	return &HTTPStatusError{Status: status, Err: err}
}

// IsNotFound returns true if the error is `sql.ErrNoRows` or `os.ErrNotExist`.
func IsNotFound(err error) bool {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return true
	case errors.Is(err, os.ErrNotExist):
		return true
	default:
		return false
	}
}

// CheckinRequest is a simple adapter that takes the raw check-in bodyBytes
// and dispatches to the respective check-in method on svc.
func CheckinRequest(svc Checkin, r *mdm.Request, bodyBytes []byte) ([]byte, error) {
	msg, err := mdm.DecodeCheckin(bodyBytes)
	if err != nil {
		return nil, NewHTTPStatusError(http.StatusBadRequest, fmt.Errorf("decoding check-in: %w", err))
	}
	var respBytes []byte
	switch m := msg.(type) {
	case *mdm.Authenticate:
		err = svc.Authenticate(r, m)
		if err != nil {
			err = fmt.Errorf("authenticate service: %w", err)
		}
	case *mdm.TokenUpdate:
		err = svc.TokenUpdate(r, m)
		if err != nil {
			err = fmt.Errorf("tokenupdate service: %w", err)
		}
	case *mdm.CheckOut:
		err = svc.CheckOut(r, m)
		if err != nil {
			err = fmt.Errorf("checkout service: %w", err)
		}
	case *mdm.UserAuthenticate:
		respBytes, err = svc.UserAuthenticate(r, m)
	case *mdm.SetBootstrapToken:
		err = svc.SetBootstrapToken(r, m)
		if err != nil {
			err = fmt.Errorf("setbootstraptoken service: %w", err)
		}
	case *mdm.GetBootstrapToken:
		// https://developer.apple.com/documentation/devicemanagement/get_bootstrap_token
		var bsToken *mdm.BootstrapToken
		bsToken, err = svc.GetBootstrapToken(r, m)
		if err != nil && !IsNotFound(err) {
			return nil, fmt.Errorf("getbootstraptoken service: %w", err)
		}
		if IsNotFound(err) || bsToken == nil {
			// Apple docs say the server should return no data and no error
			// if bootstrap token is not available
			return []byte{}, nil
		}
		respBytes, err = plist.Marshal(bsToken)
		if err != nil {
			err = fmt.Errorf("marshal bootstrap token: %w", err)
		}
	case *mdm.DeclarativeManagement:
		respBytes, err = svc.DeclarativeManagement(r, m)
		if err != nil {
			err = fmt.Errorf("declarativemanagement service: %w", err)
		}
	case *mdm.GetToken:
		if err := m.Validate(); err != nil {
			return nil, fmt.Errorf("gettoken validate: %w", err)
		}
		resp, err := svc.GetToken(r, m)
		if err != nil {
			return nil, fmt.Errorf("gettoken service: %w", err)
		}
		if resp == nil {
			return nil, errors.New("gettoken service: no response")
		}
		respBytes, err = plist.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("gettoken marshal: %w", err)
		}
		return respBytes, nil
	default:
		return nil, errors.New("unhandled check-in request type")
	}
	return respBytes, err
}

// CommandAndReportResultsRequest is a simple adapter that takes the raw
// command result report bodyBytes, dispatches to svc, and returns the
// response.
func CommandAndReportResultsRequest(svc CommandAndReportResults, r *mdm.Request, bodyBytes []byte) ([]byte, error) {
	report, err := mdm.DecodeCommandResults(bodyBytes)
	if err != nil {
		return nil, NewHTTPStatusError(http.StatusBadRequest, fmt.Errorf("decoding command results: %w", err))
	}
	cmd, err := svc.CommandAndReportResults(r, report)
	if err != nil {
		return nil, fmt.Errorf("command and report results service: %w", err)
	}
	if cmd != nil {
		return cmd.Raw, nil
	}
	return nil, nil
}
