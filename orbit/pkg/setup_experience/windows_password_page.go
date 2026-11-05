package setupexperience

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

//go:embed windows_password_page.html
var passwordPageHTML string

var passwordPageTemplate = template.Must(template.New("password").Parse(passwordPageHTML))

// errPasswordPolicy means the chosen password doesn't meet the local password policy.
var errPasswordPolicy = errors.New("password does not meet the password policy")

// passwordPage serves the "Choose your password" page shown in the setup
// experience kiosk. It listens only on loopback and requires a random token in
// the URL, so the password goes straight from the kiosk to this process and
// never reaches Fleet.
type passwordPage struct {
	minLength int
	token     string
	// create makes the account with the chosen password. It's called at most
	// once successfully; a policy error lets the user try again.
	create func(password string) error

	listener net.Listener
	server   *http.Server

	mu      sync.Mutex
	created bool
	done    chan struct{}
}

func newPasswordPage(minLength int, create func(string) error) (*passwordPage, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generating password page token: %w", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listening for password page: %w", err)
	}
	p := &passwordPage{
		minLength: minLength,
		token:     hex.EncodeToString(b),
		create:    create,
		listener:  ln,
		done:      make(chan struct{}),
	}
	p.server = &http.Server{Handler: p, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second}
	go func() {
		if err := p.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("windows setup experience: password page server")
		}
	}()
	return p, nil
}

// URL is the page address to open in the kiosk.
func (p *passwordPage) URL() string {
	return fmt.Sprintf("http://%s/?t=%s", p.listener.Addr(), p.token)
}

// Done is closed once the account has been created.
func (p *passwordPage) Done() <-chan struct{} { return p.done }

func (p *passwordPage) Close() error { return p.server.Close() }

type passwordPageData struct {
	MinLength int
	Token     string
	Error     string
	Done      bool
}

func (p *passwordPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Reject other hosts so a web page can't reach the server through DNS rebinding.
	if r.Host != p.listener.Addr().String() || r.URL.Path != "/" ||
		subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("t")), []byte(p.token)) != 1 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; form-action 'self'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := passwordPageData{MinLength: p.minLength, Token: p.token}
	switch r.Method {
	case http.MethodGet:
		p.mu.Lock()
		data.Done = p.created
		p.mu.Unlock()
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		data.Done, data.Error = p.submit(r.PostForm.Get("password"), r.PostForm.Get("confirm"))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := passwordPageTemplate.Execute(w, data); err != nil {
		log.Error().Err(err).Msg("windows setup experience: rendering password page")
	}
}

// submit creates the account and returns whether it's done, or the message to show.
func (p *passwordPage) submit(password, confirm string) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.created {
		return true, ""
	}
	switch {
	case password == "":
		return false, "Enter a password."
	case password != confirm:
		return false, "The passwords don't match."
	case len([]rune(password)) < p.minLength:
		return false, fmt.Sprintf("Use at least %d characters.", p.minLength)
	}
	if err := p.create(password); err != nil {
		if errors.Is(err, errPasswordPolicy) {
			return false, "That password doesn't meet this computer's requirements. Try a longer password with a mix of letters, numbers, and symbols."
		}
		log.Error().Err(err).Msg("windows setup experience: creating local account")
		return false, "We couldn't create your account. Try again, or contact your IT team if this keeps happening."
	}
	p.created = true
	close(p.done)
	return true, ""
}
