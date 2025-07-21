package httpsig

import (
	"context"
	"net/http"
)

type contextKey int

const verifyKey contextKey = 0

// NewSigningHTTPClient creates an *http.Client that signs requests before sending. If hc is nil a new *http.Client is created. If signer is not nil all requests will be signed. If verifier is not nil all requests will be verified.
func NewHTTPClient(hc *http.Client, signer *Signer, verifier *Verifier) *http.Client {
	if hc == nil {
		hc = &http.Client{}
	}

	hc.Transport = NewTransport(hc.Transport, signer, verifier)
	return hc
}

// NewTransport returns an http.RoundTripper implementation that signs requests and verifies responses if signer and verifier are not nil. If rt is nil http.DefaultTransport is used.
func NewTransport(rt http.RoundTripper, signer *Signer, verifier *Verifier) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &transport{
		sign:     signer != nil,
		signer:   signer,
		verify:   verifier != nil,
		verifier: verifier,
		rt:       rt,
	}
}

// VerifyHandler verifies the http signature of each request. If not verified it returns a 401 Unauthorized HTTP error. If verified it puts the verification result in the requests context. Use GetVerifyResult to read the context.
type VerifyHandler struct {
	handler  http.Handler
	verifier *Verifier
}

// NewHandler wraps an http.Handler with a an http.Handler that verifies each request. verifier cannot be nil.
func NewHandler(h http.Handler, verifier *Verifier) http.Handler {
	if verifier == nil {
		panic("verifier cannot be nil")
	}
	return &VerifyHandler{
		handler:  h,
		verifier: verifier,
	}
}

func (vh VerifyHandler) ServeHTTP(rw http.ResponseWriter, inReq *http.Request) {
	vr, err := vh.verifier.Verify(inReq)
	if err != nil {
		// Failed to verify
		rw.Write([]byte("Unauthorized"))
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	req := inReq.WithContext(context.WithValue(inReq.Context(), verifyKey, &vr))
	vh.handler.ServeHTTP(rw, req)
}

// GetVerifyResult returns the results of a successful request signature verification.
func GetVerifyResult(ctx context.Context) (v VerifyResult, found bool) {
	if vr, ok := ctx.Value(verifyKey).(*VerifyResult); ok && vr != nil {
		return *vr, true
	}
	return VerifyResult{}, false
}

// transport implements http.RoundTripper interface. It signs the request before calling the underlying RoundTripper
type transport struct {
	sign     bool
	verify   bool
	signer   *Signer
	verifier *Verifier
	rt       http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.sign {
		// Signing does not read or close the body
		err := t.signer.Sign(req)
		if err != nil {
			return nil, err
		}
	}

	resp, err := t.rt.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if t.verify {
		// Verifying does not read or close the response body
		_, err = t.verifier.VerifyResponse(resp)
	}
	return resp, err
}
