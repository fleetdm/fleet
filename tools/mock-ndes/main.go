// Command mock-ndes is a minimal mock of a Microsoft NDES (Network Device
// Enrollment Service) server, intended for exercising Fleet's NDES SCEP proxy
// end-to-end without a real Windows Server / Active Directory Certificate
// Services deployment.
//
// It emulates the two surfaces Fleet talks to:
//
//  1. The NDES admin page (default path /certsrv/mscep_admin/) which Fleet
//     scrapes over HTTP(S) with Basic/NTLM auth to obtain a one-time SCEP
//     challenge password. The page is rendered as UTF-16 LE HTML, exactly like
//     a real Windows NDES server, containing the line:
//     "The enrollment challenge password is: <B> XXXXXXXX </B>".
//
//  2. The SCEP endpoint (default path /certsrv/mscep/mscep.dll) which answers
//     the GetCACaps, GetCACert and PKIOperation operations. PKIOperation issues
//     a real certificate signed by a CA generated (or loaded) at startup, using
//     Fleet's own SCEP server implementation, so the full enrollment round-trip
//     can be tested.
//
// Every request and response is logged verbosely so the SCEP/NDES conversation
// is easy to follow.
//
// This is a development/testing tool only. Do NOT use it as a real CA.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/challenge"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
)

const (
	defaultAdminPath = "/certsrv/mscep_admin/"
	defaultSCEPPath  = "/certsrv/mscep/mscep.dll"

	// Strings Fleet's proxy looks for to surface specific NDES errors. See
	// ee/server/service/scep/scep_proxy.go.
	cacheFullMessage           = "The password cache is full."
	insufficientPermissionsMsg = "You do not have sufficient permission to enroll with SCEP."
)

func main() {
	var (
		addr       = flag.String("addr", ":8099", "address to listen on")
		username   = flag.String("username", "admin", "admin page Basic-auth username (empty disables auth)")
		password   = flag.String("password", "password", "admin page Basic-auth password (empty disables auth)")
		staticChal = flag.String("challenge", "", "use this fixed challenge password instead of generating a fresh one-time password per request")
		reqChal    = flag.Bool("require-challenge", true, "validate the SCEP challenge password against passwords handed out by the admin page")
		oneTime    = flag.Bool("one-time-challenge", true, "consume each generated challenge after a single successful PKIOperation (ignored when -challenge is set)")
		ttl        = flag.Duration("challenge-ttl", 60*time.Minute, "how long a generated challenge password stays valid")
		adminMode  = flag.String("admin-mode", "password", `admin page behavior: "password", "cache-full", "insufficient-permissions", or "empty"`)
		charset    = flag.String("charset", "utf16", `admin page encoding: "utf16" (like real Windows NDES) or "utf8"`)
		caDir      = flag.String("ca-dir", "", "directory to persist the CA (ca.pem/ca.key); a temp dir is created when empty")
		validity   = flag.Int("validity-days", 365, "validity period in days for issued certificates")
		tlsEnable  = flag.Bool("tls", false, "serve HTTPS instead of HTTP (real NDES is HTTPS)")
		tlsCert    = flag.String("tls-cert", "", "TLS certificate PEM file (a self-signed cert is generated when empty and -tls is set)")
		tlsKey     = flag.String("tls-key", "", "TLS key PEM file")
		jsonLog    = flag.Bool("json", false, "emit logs as JSON instead of text")
	)
	flag.Parse()

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	if *jsonLog {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	logger := slog.New(handler)
	ctx := context.Background()

	// CA + depot setup. We reuse Fleet's file depot and signer so PKIOperation
	// issues real certificates.
	dir := *caDir
	if dir == "" {
		var err error
		dir, err = os.MkdirTemp("", "mock-ndes-ca-")
		if err != nil {
			logger.ErrorContext(ctx, "creating temp CA dir", "err", err)
			os.Exit(1)
		}
	}
	caCert, err := ensureCA(dir)
	if err != nil {
		logger.ErrorContext(ctx, "setting up CA", "err", err)
		os.Exit(1)
	}

	depot, err := filedepot.NewFileDepot(dir)
	if err != nil {
		logger.ErrorContext(ctx, "creating file depot", "err", err)
		os.Exit(1)
	}
	caCerts, caKey, err := depot.CA(nil)
	if err != nil {
		logger.ErrorContext(ctx, "loading CA from depot", "err", err)
		os.Exit(1)
	}

	signer := scepserver.SignCSRAdapter(scepdepot.NewSigner(
		depot,
		scepdepot.WithValidityDays(*validity),
		scepdepot.WithAllowRenewalDays(14),
	))

	store := &challengeStore{
		issued:  make(map[string]time.Time),
		static:  *staticChal,
		oneTime: *oneTime,
		ttl:     *ttl,
		logger:  logger,
	}

	var csrSigner scepserver.CSRSignerContext = signer
	if *reqChal {
		csrSigner = challenge.Middleware(store, signer)
		logger.InfoContext(ctx, "SCEP challenge validation is ENABLED; PKIOperation requires a password handed out by the admin page")
	} else {
		logger.WarnContext(ctx, "SCEP challenge validation is DISABLED; PKIOperation will sign any CSR")
	}

	scepSvc, err := scepserver.NewService(caCerts[0], caKey, csrSigner)
	if err != nil {
		logger.ErrorContext(ctx, "creating SCEP service", "err", err)
		os.Exit(1)
	}
	scepHandler := scepserver.MakeHTTPHandler(scepserver.MakeServerEndpoints(scepSvc), scepSvc, logger)

	admin := &adminHandler{
		username: *username,
		password: *password,
		mode:     *adminMode,
		charset:  *charset,
		store:    store,
		logger:   logger,
	}

	mux := http.NewServeMux()
	// ServeMux longest-prefix match routes both /certsrv/mscep/mscep.dll and the
	// Windows pkiclient.exe variant to the SCEP handler, which reads the
	// operation from the query string regardless of path.
	mux.Handle("/certsrv/mscep/", scepHandler)
	mux.Handle(defaultAdminPath, admin)
	mux.Handle("/certsrv/mscep_admin", admin) // tolerate missing trailing slash
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "mock-ndes: not found; use "+defaultSCEPPath+" or "+defaultAdminPath, http.StatusNotFound)
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           loggingMiddleware(logger, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	scheme := "http"
	if *tlsEnable {
		scheme = "https"
	}
	printBanner(ctx, logger, scheme, *addr, admin, dir, caCert, *reqChal, *staticChal)

	if *tlsEnable {
		certFile, keyFile := *tlsCert, *tlsKey
		if certFile == "" || keyFile == "" {
			certFile, keyFile, err = writeSelfSignedTLS(dir)
			if err != nil {
				logger.ErrorContext(ctx, "generating self-signed TLS cert", "err", err)
				os.Exit(1)
			}
			logger.InfoContext(ctx, "generated self-signed TLS cert", "cert", certFile, "key", keyFile)
		}
		err = srv.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		logger.ErrorContext(ctx, "server stopped", "err", err)
		os.Exit(1)
	}
}

// adminHandler emulates the NDES /certsrv/mscep_admin/ page.
type adminHandler struct {
	username string
	password string
	mode     string
	charset  string
	store    *challengeStore
	logger   *slog.Logger
}

func (h *adminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Auth is enforced only when both credentials are set; emptying either one
	// disables it, matching the -username/-password flag docs.
	if h.username != "" && h.password != "" {
		user, pass, ok := r.BasicAuth()
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(h.username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(h.password)) == 1
		if !ok || !userMatch || !passMatch {
			h.logger.WarnContext(ctx, "admin page auth failed",
				"provided_basic_auth", ok, "provided_user", user)
			// Offer Basic auth. Fleet's client uses an NTLM negotiator with
			// AllowBasicAuth, so a Basic challenge is sufficient.
			w.Header().Set("WWW-Authenticate", `Basic realm="NDES"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	switch h.mode {
	case "cache-full":
		h.logger.InfoContext(ctx, "admin page returning cache-full page")
		h.write(w, ctx, cacheFullPage())
		return
	case "insufficient-permissions":
		h.logger.InfoContext(ctx, "admin page returning insufficient-permissions page")
		h.write(w, ctx, insufficientPermissionsPage())
		return
	case "empty":
		h.logger.InfoContext(ctx, "admin page returning empty body")
		w.WriteHeader(http.StatusOK)
		return
	}

	pw := h.store.issue()
	// This is a local mock NDES server used only for debugging the SCEP/NDES
	// challenge flow, so we intentionally log the challenge values (issued,
	// accepted, rejected) to make correlating requests easy.
	h.logger.InfoContext(ctx, "admin page issued SCEP challenge password", "challenge", pw)
	h.write(w, ctx, passwordPage(pw, h.store.ttl))
}

func (h *adminHandler) write(w http.ResponseWriter, ctx context.Context, html string) {
	var body []byte
	if h.charset == "utf8" {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		body = []byte(html)
	} else {
		// Real Windows NDES serves UTF-16 LE, frequently without a charset in
		// the Content-Type header and without a BOM. Fleet detects this.
		w.Header().Set("Content-Type", "text/html")
		body = utf16LE(html)
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		h.logger.WarnContext(ctx, "writing admin page body", "err", err)
	}
}

// challengeStore implements challenge.Validator. It tracks one-time passwords
// handed out by the admin page so PKIOperation can validate them, mimicking
// NDES' single-use challenge behavior.
type challengeStore struct {
	mu      sync.Mutex
	issued  map[string]time.Time
	static  string
	oneTime bool
	ttl     time.Duration
	logger  *slog.Logger
}

// issue returns a challenge password and records it as valid.
func (s *challengeStore) issue() string {
	if s.static != "" {
		return s.static
	}
	pw := randomChallenge()
	s.mu.Lock()
	s.issued[pw] = time.Now()
	s.mu.Unlock()
	return pw
}

// HasChallenge validates pw, consuming it when one-time mode is enabled.
func (s *challengeStore) HasChallenge(pw string) (bool, error) {
	// HasChallenge implements the fixed challenge.Validator interface (no ctx
	// param), so we use a background context for the structured logger.
	ctx := context.Background()
	// Local mock NDES server: log challenge values to aid debugging (see issue()).
	if s.static != "" {
		valid := subtle.ConstantTimeCompare([]byte(pw), []byte(s.static)) == 1
		s.logger.InfoContext(ctx, "validating SCEP challenge (static)", "challenge", pw, "valid", valid)
		return valid, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	issuedAt, ok := s.issued[pw]
	switch {
	case !ok:
		s.logger.WarnContext(ctx, "SCEP challenge rejected: unknown password", "challenge", pw)
		return false, nil
	case time.Since(issuedAt) > s.ttl:
		delete(s.issued, pw)
		s.logger.WarnContext(ctx, "SCEP challenge rejected: expired", "challenge", pw, "age", time.Since(issuedAt).String())
		return false, nil
	}
	if s.oneTime {
		delete(s.issued, pw)
	}
	s.logger.InfoContext(ctx, "SCEP challenge accepted", "challenge", pw, "one_time", s.oneTime)
	return true, nil
}

func randomChallenge() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// rand.Read on a healthy system does not fail; fall back to a constant.
		return "0000000000000000"
	}
	return fmt.Sprintf("%X", b)
}

// loggingMiddleware logs every request and response.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"content_length", r.ContentLength,
		}
		if op := r.URL.Query().Get("operation"); op != "" {
			attrs = append(attrs, "scep_operation", op)
		}
		if _, _, ok := r.BasicAuth(); ok {
			attrs = append(attrs, "has_basic_auth", true)
		}
		if a := r.Header.Get("Authorization"); a != "" {
			attrs = append(attrs, "authorization_scheme", schemeOf(a))
		}
		logger.InfoContext(r.Context(), "--> request", attrs...)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		logger.InfoContext(r.Context(), "<-- response",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"content_type", rec.Header().Get("Content-Type"),
			"duration", time.Since(start).String(),
		)
	})
}

func schemeOf(authHeader string) string {
	for i, c := range authHeader {
		if c == ' ' {
			return authHeader[:i]
		}
	}
	return authHeader
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wrote = true
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// ensureCA generates a self-signed CA in dir (ca.pem/ca.key) if one is not
// already present, and returns the CA certificate.
func ensureCA(dir string) (*x509.Certificate, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating CA dir: %w", err)
	}
	certPath := filepath.Join(dir, "ca.pem")
	keyPath := filepath.Join(dir, "ca.key")

	if certPEM, err := os.ReadFile(certPath); err == nil {
		if _, statErr := os.Stat(keyPath); statErr == nil {
			block, _ := pem.Decode(certPEM)
			if block == nil {
				return nil, fmt.Errorf("decoding existing %s", certPath)
			}
			return x509.ParseCertificate(block.Bytes)
		}
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Mock NDES Root CA", Organization: []string{"Fleet Mock NDES"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("creating CA certificate: %w", err)
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil { // nolint:gosec
		return nil, fmt.Errorf("writing ca.pem: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, fmt.Errorf("writing ca.key: %w", err)
	}
	return x509.ParseCertificate(der)
}

// writeSelfSignedTLS generates a self-signed TLS cert/key for the HTTPS
// listener and returns their file paths.
func writeSelfSignedTLS(dir string) (string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "mock-ndes"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost", "mock-ndes"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	certPath := filepath.Join(dir, "tls.pem")
	keyPath := filepath.Join(dir, "tls.key")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil { // nolint:gosec
		return "", "", err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

func printBanner(ctx context.Context, logger *slog.Logger, scheme, addr string, admin *adminHandler, caDir string, caCert *x509.Certificate, requireChallenge bool, staticChal string) {
	host := addr
	if len(addr) > 0 && addr[0] == ':' {
		host = "localhost" + addr
	}
	base := fmt.Sprintf("%s://%s", scheme, host)
	logger.InfoContext(ctx, "mock NDES server starting",
		"scep_url", base+defaultSCEPPath,
		"admin_url", base+defaultAdminPath,
		"admin_username", admin.username,
		// Local mock NDES debug tool: log the actual admin password and static
		// challenge so they can be used directly when reproducing SCEP flows.
		"admin_password", admin.password,
		"admin_mode", admin.mode,
		"admin_charset", admin.charset,
		"require_challenge", requireChallenge,
		"static_challenge", staticChal,
		"ca_dir", caDir,
		"ca_fingerprint_sha256", caFingerprint(caCert),
	)
}

func caFingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

// htmlPage builds an NDES-style HTML page wrapping the given body fragment.
func htmlPage(body string) string {
	return `<HTML><Head><Meta HTTP-Equiv="Content-Type" Content="text/html; charset=UTF-8">` +
		`<Title>Network Device Enrollment Service</Title></Head><Body BgColor=#FFFFFF>` +
		`<Font ID=locPageFont Face="Arial">` +
		`<P ID=locPageTitle> Network Device Enrollment Service allows you to obtain certificates ` +
		`for routers or other network devices using the Simple Certificate Enrollment Protocol (SCEP). </P>` +
		body +
		`</Font></Body></HTML>`
}

func passwordPage(pw string, ttl time.Duration) string {
	return htmlPage(
		`<P> The thumbprint (hash value) for the CA certificate is: <B> A656FA66 AB12B433 A2DA5CF7 CC153D9A </B> <P>` +
			` The enrollment challenge password is: <B> ` + pw + ` </B> <P>` +
			` This password can be used only once and will expire within ` + ttl.String() + `. <P>` +
			` Each enrollment requires a new challenge password. You can refresh this web page to obtain a new challenge password. </P>`,
	)
}

func cacheFullPage() string {
	return htmlPage(`<P> ` + cacheFullMessage + ` <P> Network Device Enrollment Service stores unused password for later use. ` +
		`By default, passwords are stored for 60 minutes. </P>`)
}

func insufficientPermissionsPage() string {
	return htmlPage(`<P> ` + insufficientPermissionsMsg + `  Please contact your system administrator. </P>`)
}

// utf16LE encodes s as UTF-16 little-endian, matching the wire format real
// Windows NDES servers use for the admin page (no BOM).
func utf16LE(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:], u)
	}
	return out
}
