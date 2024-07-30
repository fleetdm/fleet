package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

const (
	// HTTP header names
	ADMAuthSession        = "X-ADM-Auth-Session"
	ServerProtocolVersion = "X-Server-Protocol-Version"

	DefaultServerProtocolVersion = "3"

	SessionEndpoint = "/session"

	bodyForbidden = "FORBIDDEN"
)

// ErrMissingName is returned when an HTTP context is missing the DEP name.
var ErrMissingName = errors.New("transport: missing DEP name in HTTP request context")

// ctxKeyName is the context key for the DEP name.
type ctxKeyName struct{}

// WithName creates a new context from ctx with the DEP name associated.
func WithName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, ctxKeyName{}, name)
}

// GetName retrieves the DEP name from ctx.
func GetName(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyName{}).(string)
	return v
}

type AuthTokensRetriever interface {
	RetrieveAuthTokens(ctx context.Context, name string) (*OAuth1Tokens, error)
}

type SessionStore interface {
	SetSessionToken(context.Context, string, string) error
	GetSessionToken(context.Context, string) (string, error)
}

// sessionMap is a simple SessionStore which manages DEP authentication in a
// Go map. Note this potentially means that these DEP sessions are are not
// shared and thus the Apple DEP servers may not support multiple sessions at
// the same time.
type sessionMap struct {
	sessions map[string]string
	sync.RWMutex
}

// newSessionMap initializes a new sessionMap.
func newSessionMap() *sessionMap {
	return &sessionMap{sessions: make(map[string]string)}
}

func (s *sessionMap) SetSessionToken(_ context.Context, name, session string) error {
	s.Lock()
	defer s.Unlock()
	if session == "" {
		delete(s.sessions, name)
	} else {
		s.sessions[name] = session
	}
	return nil
}

func (s *sessionMap) GetSessionToken(_ context.Context, name string) (token string, err error) {
	s.RLock()
	defer s.RUnlock()
	token = s.sessions[name]
	return
}

// Transport is an http.RoundTripper that transparently handles Apple DEP API
// authentication and session token management. See the RoundTrip method for
// more details.
type Transport struct {
	// Wrapped transport that we call for actual HTTP RoundTripping.
	transport http.RoundTripper

	// Used for making the raw requests to the /session endpoint for
	// authentication and session token capture.
	client Doer

	tokens   AuthTokensRetriever
	sessions SessionStore

	// a cached pre-parsed URL of the /session path only (not a full URL)
	sessionURL *url.URL
}

// NewTransport creates a new Transport which wraps and calls to t for the
// actual HTTP calls. We call c for executing the authentication endpoint
// /session. The sessions are stored and retrieved using s while auth tokens
// are retrieved using tokens.
// If t is nil then http.DefaultTransport is used. If c is nil then
// http.DefaultClient is used. If s is nil then local-only session management
// is used. A panic will ensue if tokens is nil.
func NewTransport(t http.RoundTripper, c Doer, tokens AuthTokensRetriever, s SessionStore) *Transport {
	if t == nil {
		t = http.DefaultTransport
	}
	if c == nil {
		c = http.DefaultClient
	}
	if tokens == nil {
		panic("nil token retriever")
	}
	if s == nil {
		s = newSessionMap()
	}
	url, err := url.Parse(SessionEndpoint)
	if err != nil {
		// there shouldn't be a valid reason why url.Parse fails on this
		panic(err)
	}
	return &Transport{
		transport:  t,
		client:     c,
		tokens:     tokens,
		sessions:   s,
		sessionURL: url,
	}
}

// TeeReadCloser returns an io.ReadCloser that writes to w what it reads from rc.
// See also io.TeeReader as we simply wrap it under the hood here.
func TeeReadCloser(rc io.ReadCloser, w io.Writer) io.ReadCloser {
	type readCloser struct {
		io.Reader
		io.Closer
	}
	return &readCloser{io.TeeReader(rc, w), rc}
}

// RoundTrip transparently handles DEP server authentication and session token
// management. Practically speaking this means we make up to three individual
// requests for a given single request: the initial request attempt, a
// possible authentication request followed by a re-try of the original, now
// authenticated, request. Note also that we try to be helpful and inject the
// `X-Server-Protocol-Version` into the request headers if it is missing.
// See https://developer.apple.com/documentation/devicemanagement/device_assignment/authenticating_with_a_device_enrollment_program_dep_server
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	name := GetName(req.Context())
	if name == "" {
		return nil, ErrMissingName
	}

	// Apple DEP servers support differing requests and responses based on the
	// protocol version header. Try to be helpful and use the latest protocol
	// version mentioned in the docs.
	if _, ok := req.Header[ServerProtocolVersion]; !ok {
		req.Header.Set(ServerProtocolVersion, DefaultServerProtocolVersion)
	}

	// if previous requests have already authenticated try to use that session token
	session, err := t.sessions.GetSessionToken(req.Context(), name)
	if err != nil {
		return nil, fmt.Errorf("transport: retrieving session token: %w", err)
	}

	var resp *http.Response
	var reqBodyBuf *bytes.Buffer
	var roundTripped bool
	var forbidden bool
	if session != "" {
		// if we have a session token for this DEP name then try to inject it
		req.Header.Set(ADMAuthSession, session)
		if req.Body != nil && req.GetBody == nil {
			reqBodyBuf = bytes.NewBuffer(make([]byte, 0, req.ContentLength))
			// stream the body to both the wrapped transport and our buffer in case we need to retry
			req.Body = TeeReadCloser(req.Body, reqBodyBuf)
		}
		resp, err = t.transport.RoundTrip(req)
		if err != nil {
			return resp, err
		}
		roundTripped = true
	}
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		// the DEP simulator depsim showed this specific 403 Forbidden
		// "FORBIDDEN" error when you restart the simulator. this indicates,
		// I think, an expired/unknown session token but this isn't documented
		// for the DEP service. specifically test and handle this error so we
		// do not accidentally capture any other 403 errors (e.g. T&C).
		// unfortunately this means reading (and replacing) the body, which is
		// rather verbose.
		respBodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, fmt.Errorf("transport: reading response body: %w", err)
		}
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))
		if bytes.Contains(respBodyBytes, []byte(bodyForbidden)) {
			forbidden = true
		}
	}
	if session == "" || resp.StatusCode == http.StatusUnauthorized || forbidden {
		// either we have no session token yet or the DEP server doesn't like
		// our provided token. let's authenticate.
		tokens, err := t.tokens.RetrieveAuthTokens(req.Context(), name)
		if err != nil {
			return nil, fmt.Errorf("transport: retrieving auth tokens: %w", err)
		}

		// assemble the /session URL from the original request "base" URL.
		sessionURL := req.URL.ResolveReference(t.sessionURL)
		sessionReq, err := http.NewRequestWithContext(
			req.Context(),
			"GET",
			sessionURL.String(),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("transport: creating session request: %w", err)
		}

		// use the same version header from the original request (which we
		// likely set ourselves anyway)
		sessionReq.Header.Set(
			ServerProtocolVersion,
			req.Header.Get(ServerProtocolVersion),
		)

		session, err = DoAuth(t.client, sessionReq, tokens)
		if err != nil {
			return nil, err
		}

		// save our session token for use by following requests
		err = t.sessions.SetSessionToken(req.Context(), name, session)
		if err != nil {
			return nil, fmt.Errorf("transport: setting auth session token: %w", err)
		}

		// now that we've received and saved the session token let's use it
		// to actually make the (same) request.
		req.Header.Set(ADMAuthSession, session)

		// reset our body reader if needed
		if roundTripped && req.Body != nil {
			if req.GetBody != nil {
				// (ab)use the 304 redirect body cache if present
				req.Body, err = req.GetBody()
				if err != nil {
					return nil, err
				}
			} else if reqBodyBuf != nil {
				req.Body = io.NopCloser(reqBodyBuf)
			}
		}

		resp, err = t.transport.RoundTrip(req)
		if err != nil {
			return resp, err
		}
	}

	// check if the session token has changed. Apple says that the session
	// token can be updated from the server. save it if so.
	if respSession := resp.Header.Get(ADMAuthSession); respSession != "" && session != respSession {
		err = t.sessions.SetSessionToken(req.Context(), name, respSession)
		if err != nil {
			return nil, fmt.Errorf("transport: setting response session token: %w", err)
		}
	}

	return resp, nil
}
