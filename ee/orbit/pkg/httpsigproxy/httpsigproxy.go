package httpsig

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/remitly-oss/httpsig-go"
)

const (
	// We are using TLS 1.3 with ECC P-256 private key for performance.
	// Using this private key should cause TLS to use ECDHE-ECDSA cipher, which is faster due to a smaller key and lower compute cost.
	// To generate private key and cert:
	// openssl req -new -x509 \
	// -newkey ec:<(openssl ecparam -name prime256v1) \
	// -keyout ec_key.pem \
	// -out ec_cert.pem \
	// -days 8250 \
	// -nodes \
	// -subj "/CN=httpsig-proxy" \
	// -addext "subjectAltName = IP:127.0.0.1, IP:::1"

	// serverCert is the certificate used by the proxy server to connect to osquery via 127.0.0.1.
	serverCert = `-----BEGIN CERTIFICATE-----
MIIBqTCCAU6gAwIBAgIUCvG0XCIQmOo/16H+G4pE3tgIlg0wCgYIKoZIzj0EAwIw
GDEWMBQGA1UEAwwNaHR0cHNpZy1wcm94eTAeFw0yNTA2MjQwMzQzMTFaFw00ODAx
MjUwMzQzMTFaMBgxFjAUBgNVBAMMDWh0dHBzaWctcHJveHkwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAARJk0Q6QQYCSJamw8DUxDO8o60uU2TLa4JMJ7AEZSMX3Lc4
hwBR9WJ8bpAnvTqnF1shU01oGIOgOaH0xh84pcO+o3YwdDAdBgNVHQ4EFgQUZpLu
MKWmoOPGXmy3wkoCz/JBG5UwHwYDVR0jBBgwFoAUZpLuMKWmoOPGXmy3wkoCz/JB
G5UwDwYDVR0TAQH/BAUwAwEB/zAhBgNVHREEGjAYhwR/AAABhxAAAAAAAAAAAAAA
AAAAAAABMAoGCCqGSM49BAMCA0kAMEYCIQCypDp3B7t9Lqgxgnhl8ve2MAgiO2H4
Oq5EZgjt2ng0NwIhAKJyrItRC91gDDK2MOtWa7n8j6KjY3Kghbf4YKI/cU2l
-----END CERTIFICATE-----
`

	// serverKey is the corresponding private key. This key is compromised by
	// being in the source code, rendering any connection using this cert
	// insecure. This is OK since this connection will only be done to 127.0.0.1.
	serverKey = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg3ETz2yDl69ThBQ/o
XDL5o0YINWELb+ZJ0d5laq1ECdahRANCAARJk0Q6QQYCSJamw8DUxDO8o60uU2TL
a4JMJ7AEZSMX3Lc4hwBR9WJ8bpAnvTqnF1shU01oGIOgOaH0xh84pcO+
-----END PRIVATE KEY-----
`
)

// Proxy is the TLS proxy implementation for adding HTTP signatures. This type should only be
// initialized via NewProxy.
type Proxy struct {
	// ParsedURL is the localhost URL the proxy is listening too.
	ParsedURL       *url.URL
	CertificatePath string

	listener net.Listener
	server   *http.Server
}

// NewProxy creates a new proxy implementation targeting the provided hostname.
func NewProxy(
	proxyDirectory string,
	targetURL string,
	rootCA string,
	insecure bool,
	signer *httpsig.Signer,
) (*Proxy, error) {
	// Directory to store proxy related assets
	if err := secure.MkdirAll(proxyDirectory, constant.DefaultDirMode); err != nil {
		return nil, fmt.Errorf("there was a problem creating the proxy directory: %w", err)
	}
	// Write certificate that the local proxy will use.
	certPath := filepath.Join(proxyDirectory, "proxy.crt")
	if err := os.WriteFile(certPath, []byte(serverCert), os.FileMode(0o644)); err != nil {
		return nil, fmt.Errorf("write server cert: %w", err)
	}

	cert, err := tls.X509KeyPair([]byte(serverCert), []byte(serverKey))
	if err != nil {
		return nil, fmt.Errorf("load keypair: %w", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13, // TLS 1.3 has a faster handshake than 1.2
	}

	// Assign any available port
	listener, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		return nil, fmt.Errorf("bind 127.0.0.1: %w", err)
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("listener is not *net.TCPAddr")
	}

	handler, err := newProxyHandler(targetURL, rootCA, insecure, signer)
	if err != nil {
		return nil, fmt.Errorf("make proxy handler: %w", err)
	}

	proxy := &Proxy{
		// Rewrite URL to the proxy URL. Note the proxy handles any URL
		// prefix so we don't need to carry that over here.
		// We use 127.0.0.1 and NOT localhost due to security.
		// A misconfigured /etc/hosts could resolve localhost to something unexpected.
		ParsedURL: &url.URL{
			Scheme: "https",
			Host:   fmt.Sprintf("127.0.0.1:%d", addr.Port),
		},
		CertificatePath: certPath,
		listener:        listener,
		server: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Minute,
		},
	}

	return proxy, nil
}

// Serve will begin running the proxy.
func (p *Proxy) Serve() error {
	if p.listener == nil || p.server == nil {
		return errors.New("listener and handler must not be nil -- initialize Proxy via NewProxy")
	}
	err := p.server.Serve(p.listener)
	return fmt.Errorf("servetls returned: %w", err)
}

// Close the server and associated listener. The server may not be reused after
// calling Close().
func (p *Proxy) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return p.server.Shutdown(ctx)
}

func newProxyHandler(targetURL string, rootCA string, insecure bool, signer *httpsig.Signer) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parse target url: %w", err)
	}

	transport := fleethttp.NewTransport()
	switch {
	case insecure:
		transport.TLSClientConfig.InsecureSkipVerify = true
	case rootCA != "":
		rootCAs, err := certificate.LoadPEM(rootCA)
		if err != nil {
			return nil, fmt.Errorf("loading server root CA: %w", err)
		}
		transport.TLSClientConfig.RootCAs = rootCAs
	}

	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)
		},
		Transport: &signingRoundTripper{
			signer:    signer,
			transport: transport,
		},
	}
	return reverseProxy, nil
}

// Copied from Go source
// https://go.googlesource.com/go/+/go1.15.6/src/net/http/httputil/reverseproxy.go#114
func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()
	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")
	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

// Copied from Go source
// https://go.googlesource.com/go/+/go1.15.6/src/net/http/httputil/reverseproxy.go#102
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

type signingRoundTripper struct {
	signer    *httpsig.Signer
	transport http.RoundTripper
}

func (s *signingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Sign the request before sending
	if err := s.signer.Sign(req); err != nil {
		return nil, fmt.Errorf("signing request: %#v", err)
	}

	// Remove X-Forwarded-For because we are forwarding from 127.0.0.1,
	// which is a non-standard use of this header and may be rejected by some load balancers.
	req.Header.Del("X-Forwarded-For")

	return s.transport.RoundTrip(req)
}
