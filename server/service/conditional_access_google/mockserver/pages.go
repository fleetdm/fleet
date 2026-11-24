package mockserver

import (
	"html/template"
	"net/http"
	"time"
)

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
{{if .AutoContinue}}<meta http-equiv="refresh" content="{{.AutoContinue}};url={{.Continue}}">{{end}}
<style>body{font-family:-apple-system,system-ui,sans-serif;max-width:32rem;margin:3rem auto;padding:0 1rem;line-height:1.5}
small{color:#666}a{display:block;margin:.5rem 0}</style></head>
<body>
<h1>{{.Heading}}</h1>
{{if .Body}}<p>{{.Body}}</p>{{end}}
{{if .Message}}<p><small>{{.Message}}</small></p>{{end}}
{{if .Continue}}
<p><small>Mock Google sign-in as {{.Email}}.{{if .AutoContinue}} Continuing in {{.AutoContinue}} seconds.{{end}}</small></p>
<a id="continue" href="{{.Continue}}">Continue</a>
<a id="not-found" href="{{.NotFound}}">Continue as a user who hasn't signed in to Gmail on this device</a>
<a id="refused" href="{{.Refused}}">Fail the sign-in</a>
{{end}}
</body></html>`))

type page struct {
	Title, Heading, Body, Message string
	Email                         string
	Continue, NotFound, Refused   string
	AutoContinue                  int // seconds, 0 for never
}

func renderPage(w http.ResponseWriter, p page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTemplate.Execute(w, p)
}

func (s *Server) renderFakeSignIn(w http.ResponseWriter, state string) {
	renderPage(w, page{
		Title: "Sign in - Google Accounts (mock)", Heading: "Sign in with Google", Email: s.cfg.FakeEmail,
		AutoContinue: int(s.cfg.AutoContinue / time.Second),
		Continue:     s.callbackURL(state, CodeSignedIn),
		NotFound:     s.callbackURL(state, CodeNotFound),
		Refused:      s.callbackURL(state, CodeRefused),
	})
}

// Copy of fleetdm.com's result pages.
var resultCopy = map[string]page{
	"done":        {Heading: "You're done! 🎉", Body: "You can log in to apps with your company email."},
	"not-allowed": {Heading: "This device isn't allowed to access company data.", Body: "Only company-owned devices that are enrolled to Fleet are allowed."},
	"not-found":   {Heading: "Sign in to Gmail on this device, then open this page again."},
	"error":       {Heading: "Something went wrong. Try again."},
}

func (s *Server) handleResultPage(w http.ResponseWriter, r *http.Request) {
	p, ok := resultCopy[r.PathValue("result")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	p.Title = p.Heading
	p.Message = r.URL.Query().Get("message")
	renderPage(w, p)
}
