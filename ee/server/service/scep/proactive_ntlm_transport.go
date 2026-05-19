package scep

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/Azure/go-ntlmssp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// proactiveNTLMTransport sends the NTLM Type-1 (Negotiate) message on the
// first request, then the Type-3 (Authenticate) message on the same
// keepalive connection.
//
// We bypass ntlmssp.Negotiator's v0.1.0+ anonymous-probe step because
// NTLM is connection-bound on IIS and the extra unauthenticated round-trip
// breaks the handshake whenever the server or a fronting reverse proxy
// (Okta Access, a WAF, IIS with anonymous auth enabled at the site root)
// returns 200 to the probe or closes the connection after the 401.
type proactiveNTLMTransport struct {
	base http.RoundTripper
}

func (t *proactiveNTLMTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	username, password, ok := req.BasicAuth()
	if !ok {
		return t.base.RoundTrip(req)
	}

	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return nil, ctxerr.New(req.Context(), "ntlm: request body is not replayable; set req.GetBody")
	}

	negotiate, err := ntlmssp.NewNegotiateMessage("", "")
	if err != nil {
		return nil, ctxerr.Wrap(req.Context(), err, "ntlm: create negotiate message")
	}

	req1 := req.Clone(req.Context())
	req1.Header.Set("Authorization", "NTLM "+base64.StdEncoding.EncodeToString(negotiate))

	resp1, err := t.base.RoundTrip(req1)
	if err != nil {
		return nil, err
	}
	if resp1.StatusCode != http.StatusUnauthorized {
		return resp1, nil
	}

	var challengeB64 string
	for _, h := range resp1.Header.Values("WWW-Authenticate") {
		lower := strings.ToLower(h)
		switch {
		case strings.HasPrefix(lower, "ntlm "):
			challengeB64 = h[len("NTLM "):]
		case strings.HasPrefix(lower, "negotiate "):
			challengeB64 = h[len("Negotiate "):]
		}
		if challengeB64 != "" {
			break
		}
	}
	if challengeB64 == "" {
		// No NTLM/Negotiate challenge to act on; hand the 401 back to the
		// caller with the body intact so they can inspect it.
		return resp1, nil
	}
	_, _ = io.Copy(io.Discard, resp1.Body)
	_ = resp1.Body.Close()

	challenge, err := base64.StdEncoding.DecodeString(challengeB64)
	if err != nil {
		return nil, ctxerr.Wrap(req.Context(), err, "ntlm: decode challenge")
	}

	authenticate, err := ntlmssp.NewAuthenticateMessage(challenge, username, password, nil)
	if err != nil {
		return nil, ctxerr.Wrap(req.Context(), err, "ntlm: create authenticate message")
	}

	req2 := req.Clone(req.Context())
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, ctxerr.Wrap(req.Context(), err, "ntlm: rewind request body")
		}
		req2.Body = body
	}
	req2.Header.Set("Authorization", "NTLM "+base64.StdEncoding.EncodeToString(authenticate))

	return t.base.RoundTrip(req2)
}
