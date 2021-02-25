// Package insecure provides an insecure (if it were not obvious yet) TLS proxy
// that can be used for testing osquery enrollment with a Fleet (or other TLS)
// server in non-production environments.
//
// Functions in this package are NOT SUITABLE FOR PRODUCTION ENVIRONMENTS!
package insecure

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// ServerCert is the certificate used by the proxy server.
	ServerCert = `-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQCPnw3uINXlozANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjAxMjE5MDA0MDA0WhcNNDgwNTA1MDA0MDA0WjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC1
GuqbXIo767H1dPM9KS7Uz6bU3Jzh/4e5fxmgxTz2GY76UiKhBKvlWIy2PsFMKpQ1
kd3/MyANoOcUkdolPAX/6jMZc9qhlRaG80MqgZuBzX3KHnCnFN9vin1wOrTlyboW
NLjKCmKTCpa0knuya9hgOwCJ1cFMFByC29qRvYKtisQxRbpy/d/jN14dXsGeQiZW
KU6ncmFPBH8+uTnrQq4A3UBFMOu5C+Uk+hCSLNMu4ZbAUR41m0LpR5OaWk1t0q2O
ZbDg4zkSJzbBNeiVe+vCbKtevqtRgqAi4u4EGdasnlhEJ/UPfF9lvqCd1iRw7M9u
quPlsJs6tE4GFBpIUUMBAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAA5KnpFoTnKW
B04G42v6a2AkY/ENEgoMhKr6JBeRkRKF6Itatiotb/RClgRYlDUn+ljow8/Tyds4
qqMl/MzjbbwI4xNcu9t+0bG2zmJj6ON4mbRH+GnBPX+t50/1eKSoPjtHDyT/UAbx
q3jyXp0nObaRzDqmYK/OUVg7vhAxQqQ9Cvvk819Ar8wFZGjE9Bc2YDObyCVQWCZz
qIfzr/Qh46tq0o+KdlaV2oHy4VLrLOFXeD5MKf6A7aOP7h9Yy9ywnScrobaSXwd8
kS/PZzVeJtwvKf+c1tBiJxHix2vLiFtS5IKdhNGKNvMyQNWgq046iTNeVJkxo+Qb
YP4a5WpD+aw=
-----END CERTIFICATE-----
`

	// serverKey is the corresponding private key. This key is compromised by
	// being in the source code, rendering any connection using this cert
	// insecure.
	serverKey = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC1GuqbXIo767H1
dPM9KS7Uz6bU3Jzh/4e5fxmgxTz2GY76UiKhBKvlWIy2PsFMKpQ1kd3/MyANoOcU
kdolPAX/6jMZc9qhlRaG80MqgZuBzX3KHnCnFN9vin1wOrTlyboWNLjKCmKTCpa0
knuya9hgOwCJ1cFMFByC29qRvYKtisQxRbpy/d/jN14dXsGeQiZWKU6ncmFPBH8+
uTnrQq4A3UBFMOu5C+Uk+hCSLNMu4ZbAUR41m0LpR5OaWk1t0q2OZbDg4zkSJzbB
NeiVe+vCbKtevqtRgqAi4u4EGdasnlhEJ/UPfF9lvqCd1iRw7M9uquPlsJs6tE4G
FBpIUUMBAgMBAAECggEBAKUhQcEfA7vXEJBqbk7Z+iV4oPl9nl5CjBKK3WdF8GvE
qiV8Nq7yf3nC36pcVguI11JxCiXjC9rhV1HeGzXQIPhTJvySMksakUvDCv765jvY
jlV4o+b0lTYy5GUsYj0TTmVo9QTjqzW/deJ3nen1g3la0wbarEEeJVD7/bLdRQXN
9DcFtKpC+to0N7RE2cFLGIgWcU/W6IF+DoLkVKdd0h2EZ4FawRtythkfC2Z+B1W/
ayz645XIQcA2sI2z4G2CqR/k3gmv6eGzoQgTN46z2lYqPFMD7v8hwv9Z/LCIghF9
iJQy8Uxy8zRjmCDNP0oWRrlIUh2abqhgLlvvYsFJ3QECgYEA4Z0FJmsgf65NlBzQ
g5+x9pQLZX0tEpeuDZT+W9i+7u/d1nkISmD+urL+f1o5Jc8dg0QG9Y6XFodIsp8W
s6wbCcBR+V7q76fJONRLnh5mHNwBvvVGg5Iy/Bmc8gTboG7F4yJnrvCYeE4gIVXt
4c5c7B1hOC/eyXIGng5bbKwtCAkCgYEAzX9JSOwWlt3Y75SCL8d3WSEBVqgaH1UZ
4lSZ/Bi1F7uUToJGMnH2XinZ0tO7erPtisl/S9SZhMmIFho+p/23Mz4SWtLGzIBu
q8Zoo4gfCk2cP5YkidKad1Kc3ZG4LWQCiRr7+HYYmTXaUWe5PNznlp36IFliPlkK
Q2ztNOpq8TkCgYBa4SQ08IwbwnuPgPfhPU+zcrkQfZbNWXoMEItRNgLbPpYOkZxs
UZvqWrW3WQGSIFbUDG/9NB3aPk5jXUAIyffuOqEKoVhjhyPAF4wKOlaJo3m0kRqB
Xz/YWvzkZF6Pxm9B6hb32gSg2V+J7hIvli/KEJ+bwXStkpflzQS4xrYw+QKBgGqH
T8Bj0xoGi403WX3XU4F64KzBnDkd7rsrzF+pl0dkUG+ajTVdarBJ1ce7R3dGix/l
cP4oiiUSLF/43v5LQotn5C/9EF23PqgBxQDxcdXvgc5c0Tg5WyX8R6F9BxNQwxe8
S170KbBTAIgu0xJAGjY0UxQuAgX8NpvZfeZul13RAoGARVSBIo5MDodDw25j4fc7
YIC5ppouCT4VTz6IUOxYyw8TQBBm5Wes62JJ+9yTLLfIRNnkDkVwSX7Q3blva+1W
1lFBG1WzNhTRo0im9lRyVCGp2ZSm2UxsJS9jP1J6bkavkjK/jaPOC9J6dmWOZBWm
yJY8h/0WhN9XPcXFa4Qmfu0=
-----END PRIVATE KEY-----
`
)

// TLSProxy is the insecure TLS proxy implementation. This type should only be
// initialized via NewTLSProxy.
type TLSProxy struct {
	// Port is the port the TLS proxy is listening on (always on localhost).
	Port int

	listener net.Listener
	server   *http.Server
}

// NewTLSProxy creates a new proxy implementation targeting the provided
// hostname.
func NewTLSProxy(targetURL string) (*TLSProxy, error) {
	cert, err := tls.X509KeyPair([]byte(ServerCert), []byte(serverKey))
	if err != nil {
		return nil, errors.Wrap(err, "load keypair")
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	// Assign any available port
	listener, err := tls.Listen("tcp", "localhost:0", cfg)
	if err != nil {
		return nil, errors.Wrap(err, "bind localhost")
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, errors.New("listener is not *net.TCPAddr")
	}

	handler, err := newProxyHandler(targetURL)
	if err != nil {
		return nil, errors.Wrap(err, "make proxy handler")
	}

	proxy := &TLSProxy{
		Port:     addr.Port,
		listener: listener,
		server:   &http.Server{Handler: handler},
	}

	return proxy, nil
}

// InsecureServeTLS will begin running the TLS proxy.
func (p *TLSProxy) InsecureServeTLS() error {
	if p.listener == nil || p.server == nil {
		return errors.New("listener and handler must not be nil -- initialize TLSProxy via NewTLSProxy")
	}

	err := p.server.Serve(p.listener)
	return errors.Wrap(err, "servetls returned")
}

// Close the server and associated listener. The server may not be reused after
// calling Close().
func (p *TLSProxy) Close() error {
	// err := p.listener.Close()
	// if err != nil {
	// 	return errors.Wrap(err, "close listener")
	// }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return p.server.Shutdown(ctx)
}

func newProxyHandler(targetURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, errors.Wrap(err, "parse target url")
	}

	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)
		},
	}
	// Adapted from http.DefaultTransport
	reverseProxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
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
