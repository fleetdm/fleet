package token

import (
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	TokenValidationFunc func(string) error
}

func (s *mockClient) CheckToken(token string) error {
	return s.TokenValidationFunc(token)
}

type mockReader struct {
	ReadFunc  func() (string, error)
	CachedVal string
}

func (s *mockReader) Read() (string, error) {
	return s.ReadFunc()
}

func (s *mockReader) GetCached() string {
	return s.CachedVal
}

func TestNewChecker(t *testing.T) {
	client := &mockClient{}
	checker := NewChecker("path/to/token", client)

	require.NotNil(t, checker)
}

func TestIsValid(t *testing.T) {
	client := &mockClient{}
	checker := NewChecker("path/to/token", client)

	testCases := []struct {
		err      error
		expected bool
	}{
		{nil, true},
		{errors.New("random error"), false},
		{service.ErrMissingLicense, true},
	}

	for _, tc := range testCases {
		result := checker.isValid(tc.err)
		require.Equal(t, tc.expected, result)
	}
}

func TestAwaitValid_Success(t *testing.T) {
	client := &mockClient{
		TokenValidationFunc: func(token string) error {
			return nil
		},
	}
	reader := &mockReader{
		ReadFunc: func() (string, error) {
			return "valid token", nil
		},
	}

	checker := &RemoteChecker{
		reader: reader,
		client: client,
	}

	done := make(chan bool)
	go func() {
		checker.AwaitValid()
		done <- true
	}()

	select {
	case <-done:
		// test passed
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out - AwaitValid did not complete in expected time")
	}
}

func TestAwaitValid_Failure(t *testing.T) {
	client := &mockClient{
		TokenValidationFunc: func(token string) error {
			return errors.New("invalid token")
		},
	}
	reader := &mockReader{
		ReadFunc: func() (string, error) {
			return "invalid token", nil
		},
	}

	checker := &RemoteChecker{
		reader: reader,
		client: client,
	}

	done := make(chan bool)
	go func() {
		checker.AwaitValid()
		done <- true
	}()

	select {
	case <-done:
		t.Fatal("Test failed - AwaitValid should not have completed")
	case <-time.After(2 * time.Second):
		// test passed
	}
}

func TestAwaitValid_ReaderError(t *testing.T) {
	client := &mockClient{}
	reader := &mockReader{
		ReadFunc: func() (string, error) {
			return "", errors.New("reader error")
		},
	}

	checker := &RemoteChecker{
		reader: reader,
		client: client,
	}

	done := make(chan bool)
	go func() {
		checker.AwaitValid()
		done <- true
	}()

	select {
	case <-done:
		t.Fatal("Test failed - AwaitValid should not have completed")
	case <-time.After(2 * time.Second):
		// test passed
	}
}
