package mockserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	relayPath         = "/api/v1/google-conditional-access"
	signInPath        = relayPath + "/sign-in"
	callbackPath      = relayPath + "/callback"
	resultPathPrefix  = "/google-conditional-access/"
	fleetRegisterPath = "/api/v1/fleet/google_conditional_access/register"
	signInLifetime    = 10 * time.Minute
)

type registration struct {
	customerID string
	origin     string
	secret     string
}

type pendingSignIn struct {
	customerID string
	hostUUID   string
	token      string
	expiresAt  time.Time
}

func (s *Server) relayRoutes() {
	s.mux.HandleFunc("POST "+relayPath, s.simulate(s.handleCreateRegistration, failRelay))
	s.mux.HandleFunc("DELETE "+relayPath, s.simulate(s.handleDeleteRegistration, failRelay))
	s.mux.HandleFunc("GET "+signInPath, s.simulate(s.handleSignIn, failRelay))
	s.mux.HandleFunc("GET "+callbackPath, s.handleCallback)
	s.mux.HandleFunc("GET "+resultPathPrefix+"{result}", s.handleResultPage)
}

func (s *Server) handleCreateRegistration(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	if origin == "" {
		http.Error(w, "missing Origin header", http.StatusBadRequest)
		return
	}
	var req struct {
		CustomerID string `json:"googleCustomerId"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.CustomerID == "" {
		http.Error(w, "missing googleCustomerId", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	existing := s.registrations[req.CustomerID]
	if existing != nil && existing.origin != origin {
		s.mu.Unlock()
		http.Error(w, "customer already registered from "+existing.origin, http.StatusConflict)
		return
	}
	reg := &registration{customerID: req.CustomerID, origin: origin, secret: randomHex(15)}
	s.registrations[req.CustomerID] = reg
	s.mu.Unlock()

	s.logger.InfoContext(r.Context(), "registered fleet server", "customer_id", req.CustomerID, "origin", origin)
	writeJSON(w, http.StatusOK, map[string]string{"fleetServerSecret": reg.secret})
}

// registration returns a copy of the customer's registration, or nil.
func (s *Server) registration(customerID string) *registration {
	s.mu.Lock()
	defer s.mu.Unlock()
	reg := s.registrations[customerID]
	if reg == nil {
		return nil
	}
	cp := *reg
	return &cp
}

func (s *Server) handleDeleteRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string `json:"googleCustomerId"`
		Secret     string `json:"fleetServerSecret"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	reg := s.registrations[req.CustomerID]
	ok := reg != nil && req.Secret != "" && reg.secret == req.Secret
	if ok {
		delete(s.registrations, req.CustomerID)
	}
	s.mu.Unlock()
	if !ok {
		http.Error(w, "no registration matches the customer ID and secret", http.StatusNotFound)
		return
	}
	s.logger.InfoContext(r.Context(), "deregistered fleet server", "customer_id", req.CustomerID)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSignIn(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := pendingSignIn{
		customerID: q.Get("googleCustomerId"),
		hostUUID:   q.Get("hostUuid"),
		token:      q.Get("token"),
		expiresAt:  time.Now().Add(signInLifetime),
	}
	if p.customerID == "" || p.hostUUID == "" || p.token == "" {
		http.Error(w, "googleCustomerId, hostUuid and token are required", http.StatusBadRequest)
		return
	}
	state := randomHex(16)
	s.mu.Lock()
	_, ok := s.registrations[p.customerID]
	if ok {
		// Sign-ins that never reach the callback would pile up otherwise.
		maps.DeleteFunc(s.signIns, func(_ string, old *pendingSignIn) bool { return p.expiresAt.Sub(old.expiresAt) > signInLifetime })
		s.signIns[state] = &p
	}
	s.mu.Unlock()
	if !ok {
		http.Error(w, "no registration for this customer", http.StatusNotFound)
		return
	}

	if s.cfg.OAuth != nil {
		http.Redirect(w, r, s.cfg.OAuth.authCodeURL(s.cfg.PublicURL+callbackPath, state), http.StatusFound)
		return
	}
	s.renderFakeSignIn(w, state)
}

func (s *Server) callbackURL(state, code string) string {
	q := url.Values{}
	q.Set("state", state)
	q.Set("code", code)
	return s.cfg.PublicURL + callbackPath + "?" + q.Encode()
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	state := q.Get("state")
	s.mu.Lock()
	p := s.signIns[state]
	delete(s.signIns, state)
	s.mu.Unlock()
	if p == nil || time.Now().After(p.expiresAt) {
		s.redirectResult(w, r, "error", "The sign-in expired or was already used.")
		return
	}
	if e := q.Get("error"); e != "" {
		s.redirectResult(w, r, "error", "Google sign-in failed: "+e)
		return
	}

	lookup, err := s.lookupIOS(ctx, p, q.Get("code"))
	if err != nil {
		s.logger.ErrorContext(ctx, "ios lookup", "err", err)
		s.redirectResult(w, r, "error", err.Error())
		return
	}

	reg := s.registration(p.customerID)
	if reg == nil {
		s.redirectResult(w, r, "error", "The Fleet server is no longer registered.")
		return
	}

	result, err := s.callFleet(ctx, reg, map[string]any{
		"token": p.token, "names": nonNil(lookup.Names), "customer": lookup.Customer, "email": lookup.Email,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "fleet register call", "err", err)
		s.redirectResult(w, r, "error", err.Error())
		return
	}
	s.logger.InfoContext(ctx, "fleet register call", "host_uuid", p.hostUUID, "names", lookup.Names, "result", result)
	s.redirectResult(w, r, result, "")
}

// callFleet sends the lookup to the Fleet server and returns its result.
func (s *Server) callFleet(ctx context.Context, reg *registration, body any) (string, error) {
	base := s.cfg.FleetURL
	if base == "" {
		base = reg.origin
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+fleetRegisterPath, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+reg.secret)
	resp, err := s.cfg.FleetClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call the Fleet server: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("the Fleet server answered %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var out struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("decode the Fleet server's answer: %w", err)
	}
	return out.Result, nil
}

var resultPages = map[string]string{
	"done":        "done",
	"not_allowed": "not-allowed",
	"not_found":   "not-found",
	"error":       "error",
}

func (s *Server) redirectResult(w http.ResponseWriter, r *http.Request, result, message string) {
	page, ok := resultPages[result]
	if !ok {
		page, message = "error", "unexpected result "+result
	}
	target := resultPathPrefix + page
	if message != "" {
		target += "?message=" + url.QueryEscape(message)
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
