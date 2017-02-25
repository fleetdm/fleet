package license

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

var tokenString = "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsImtpZCI6IjRkOmM1OmRlOmE1Oj" +
	"czOmUxOmE4OjI4OmU2OmEyOjMwOmI4OmI1OjBmOjg4OjQ0In0.eyJsaWNlbnNlX3V1aWQiOiIyZD" +
	"gwMmEyYS1hZjRjLTQ5ZjItYWRlNC0zOGJmNjBmMmQxZjYiLCJvcmdhbml6YXRpb25fbmFtZSI6Il" +
	"BoYW50YXNtLCBJbmMuIiwib3JnYW5pemF0aW9uX3V1aWQiOiI5ZmFiNjdiMy0wZWFjLTRhODMtOTI" +
	"wNS04MjkyMWIwNDJmODYiLCJob3N0X2xpbWl0IjowLCJldmFsdWF0aW9uIjp0cnVlLCJleHBpcmV" +
	"zX2F0IjoiMjAxNy0wMy0wNFQxNDozODo1NyswMDowMCJ9.DRFQIUDFXT0bDdya0IJKvATKCJjv3Mv" +
	"w5gMxHNzby_L80muoe-36DoRxBAJZHL7dOfQDU8NRK2Mt64ozThrhWVl8wJlD9mk5ABe3tNw3LJRl" +
	"2mHvOLmk37_AIHp5AEKZ6cWMPa9zf8hWf6bAv_0rOJf5wgyE81pfqRFtO0OnkGO3WLcP66L0AIntq" +
	"IzAE_vWmizcUvUOCWDqwcBlT-P1mZnWJFCaSBpmpQoi3KEKJDx0wMjLiRNLX9R9dr3v3ojccoYuxR" +
	"qAws-OHv3VzcuGdn3Pt9WBDr4cXdtqxaGtxJb6-BDvp8QQk69ACZXrZJ8NhZAL0EVlviRRw8bbEYchZQ"

var publicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0ZhY7r6HmifXPtServt4
D3MSi8Awe9u132vLf8yzlknvnq+8CSnOPSSbCD+HajvZ6dnNJXjdcAhuZ32ShrH8
rEQACEUS8Mh4z8Mo5Nlq1ou0s2JzWCx049kA34jP3u6AiPgpWUf8JRGstTlisxMn
H6B7miDs1038gVbN5rk+j+3ALYzllaTnCX3Y0C7f6IW7BjNO/tvFB84/95xfOLEz
o2MeFMqkD29hvcrUW+8+fQGJaVLvcEqBDnIEVbCCk8Wnoi48dUE06WHUl6voJecD
dW1E6jHcq8PQFK+4bI1gKZVbV4dFGSSMUyD7ov77aWHjxdQe6YEGcSXKzfyMaUtQ
vQIDAQAB
-----END PUBLIC KEY-----
`

func mockTicker(ticker clock.Ticker) Option {
	return func(chk *Checker) {
		chk.ticker = ticker
	}
}

func TestLicenseFound(t *testing.T) {
	var licFunInvoked int64
	var revokeFunInvoked int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := revokeInfo{
			UUID:    "DEADBEEF",
			Revoked: true,
		}
		json.NewEncoder(w).Encode(response)

	}))
	defer ts.Close()

	ds := new(mock.Store)
	ds.LicenseFunc = func() (*kolide.License, error) {
		atomic.AddInt64(&licFunInvoked, 1)
		result := &kolide.License{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now().Add(-5 * time.Minute),
			},
			Token:     &tokenString,
			PublicKey: publicKey,
			Revoked:   false,
			ID:        1,
		}
		return result, nil
	}
	ds.RevokeLicenseFunc = func(revoked bool) error {
		atomic.AddInt64(&revokeFunInvoked, 1)
		return nil
	}
	c := clock.NewMockClock()
	checker := NewChecker(ds, ts.URL,
		mockTicker(c.NewTicker(time.Millisecond)),
	)
	checker.Start()
	checker.RunLicenseCheck(context.Background())
	<-time.After(10 * time.Millisecond)
	c.AddTime(time.Millisecond)
	c.AddTime(time.Millisecond)
	<-time.After(10 * time.Millisecond)
	checker.Stop()

	// verify muliple checks occurred, we have to use atomic because if we
	// use the  flags from the mock package to indicate function invocation race detector will
	// complain

	assert.Equal(t, int64(3), atomic.LoadInt64(&licFunInvoked))
	assert.Equal(t, int64(3), atomic.LoadInt64(&revokeFunInvoked))
}

func TestLicenseNotFound(t *testing.T) {
	var licFunInvoked int64
	var revokeFunInvoked int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		response := revokeError{
			Status: 404,
			Error:  "not found",
		}
		json.NewEncoder(w).Encode(response)

	}))
	defer ts.Close()

	ds := new(mock.Store)
	ds.LicenseFunc = func() (*kolide.License, error) {
		atomic.AddInt64(&licFunInvoked, 1)
		result := &kolide.License{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now().Add(-5 * time.Minute),
			},
			Token:     &tokenString,
			PublicKey: publicKey,
			Revoked:   false,
			ID:        1,
		}
		return result, nil
	}
	ds.RevokeLicenseFunc = func(revoked bool) error {
		atomic.AddInt64(&revokeFunInvoked, 1)
		return nil
	}

	c := clock.NewMockClock()
	checker := NewChecker(ds, ts.URL,
		mockTicker(c.NewTicker(time.Millisecond)),
	)
	checker.Start()
	checker.RunLicenseCheck(context.Background())
	<-time.After(10 * time.Millisecond)
	c.AddTime(time.Millisecond)
	<-time.After(10 * time.Millisecond)
	checker.Stop()

	assert.Equal(t, int64(2), atomic.LoadInt64(&licFunInvoked))
	assert.Equal(t, int64(0), atomic.LoadInt64(&revokeFunInvoked))
}

type testLogger struct {
	logContent string
	lock       sync.Mutex
}

func (tl *testLogger) Log(keyVals ...interface{}) error {
	tl.lock.Lock()
	defer tl.lock.Unlock()
	tl.logContent += fmt.Sprint(keyVals...)
	return nil
}

func (tl *testLogger) read() string {
	var buff []byte
	tl.lock.Lock()
	buff = make([]byte, len(tl.logContent))
	copy(buff, tl.logContent)
	tl.lock.Unlock()
	return string(buff)
}

func TestLicenseTimeout(t *testing.T) {
	var licFunInvoked int64
	var revokeFunInvoked int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-time.After(300 * time.Millisecond)
		response := revokeInfo{
			UUID:    "DEADBEEF",
			Revoked: true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	ds := new(mock.Store)
	ds.LicenseFunc = func() (*kolide.License, error) {
		atomic.AddInt64(&licFunInvoked, 1)
		result := &kolide.License{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: time.Now().Add(-5 * time.Minute),
			},
			Token:     &tokenString,
			PublicKey: publicKey,
			Revoked:   false,
			ID:        1,
		}
		return result, nil
	}
	ds.RevokeLicenseFunc = func(revoked bool) error {
		atomic.AddInt64(&revokeFunInvoked, 1)
		return nil
	}

	// inject our custom logger so we can get log without breaking race
	// detection
	logger := &testLogger{}
	c := clock.NewMockClock()

	checker := NewChecker(ds, ts.URL,
		mockTicker(c.NewTicker(time.Millisecond)),
		HTTPClient(&http.Client{Timeout: 2 * time.Millisecond}),
		Logger(logger),
	)
	checker.Start()
	checker.RunLicenseCheck(context.Background())
	<-time.After(10 * time.Millisecond)
	c.AddTime(time.Millisecond)
	<-time.After(10 * time.Millisecond)
	checker.Stop()

	assert.Equal(t, int64(2), atomic.LoadInt64(&licFunInvoked))
	assert.Equal(t, int64(0), atomic.LoadInt64(&revokeFunInvoked))
	match, _ := regexp.MatchString("(Client.Timeout exceeded while awaiting headers)", logger.read())
	assert.True(t, match)
	// check to make sure things cleanly shut down.
	match, _ = regexp.MatchString("finishing", logger.read())
	assert.True(t, match)

}

func TestURLAddVersionInfo(t *testing.T) {
	licenseURL := "https://kolide.co/api/v0/licenses"
	ur, err := addVersionInfo(licenseURL)
	require.Nil(t, err)
	want := licenseURL + "?version=unknown"
	assert.Equal(t, want, ur.String(), "query params must include version")
}
