// Package server implements the ACME server (RFC 8555) that faces devices.
//
// It manages the full ACME lifecycle (accounts, orders, authorizations,
// challenges, nonces) and delegates certificate issuance and challenge
// validation to a pluggable CertificateIssuer backend.
//
// For the POC, JWS validation is skipped — request payloads are accepted
// as plain JSON. This will be added in production.
package server

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
)

const (
	// PathPrefix is the URL path prefix for all ACME server endpoints.
	PathPrefix = "/api/mdm/acme/"
)

// Server is the ACME server that faces devices.
type Server struct {
	store   *Store
	issuers map[string]api.CertificateIssuer // caName -> issuer
	baseURL string                           // e.g., "https://fleet.example.com"
	logger  *slog.Logger
}

// New creates a new ACME server.
// baseURL is the Fleet server's external URL (e.g., "https://fleet.example.com").
func New(baseURL string, logger *slog.Logger) *Server {
	return &Server{
		store:   NewStore(),
		issuers: make(map[string]api.CertificateIssuer),
		baseURL: strings.TrimSuffix(baseURL, "/"),
		logger:  logger,
	}
}

// RegisterIssuer registers a CertificateIssuer backend for the given CA name.
func (s *Server) RegisterIssuer(caName string, issuer api.CertificateIssuer) {
	s.issuers[caName] = issuer
}

// ServeHTTP implements http.Handler. Routes requests based on {ca} path segment.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip prefix to get "/{ca}/..."
	path := strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(PathPrefix, "/"))
	if path == r.URL.Path {
		http.NotFound(w, r)
		return
	}

	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/", 2)
	caName := parts[0]
	if caName == "" {
		http.NotFound(w, r)
		return
	}

	var subPath string
	if len(parts) > 1 {
		subPath = "/" + parts[1]
	} else {
		subPath = "/"
	}

	if _, ok := s.issuers[caName]; !ok {
		s.logger.Warn("unknown CA", "ca", caName)
		s.writeACMEError(w, http.StatusNotFound, "urn:ietf:params:acme:error:malformed",
			fmt.Sprintf("unknown CA: %s", caName))
		return
	}

	// Add nonce to every response
	w.Header().Set("Replay-Nonce", s.store.CreateNonce())

	s.logger.Info("ACME request", "ca", caName, "method", r.Method, "path", subPath)

	switch {
	case subPath == "/directory":
		s.handleDirectory(w, r, caName)
	case subPath == "/new-nonce":
		s.handleNewNonce(w, r)
	case subPath == "/new-account":
		s.handleNewAccount(w, r, caName)
	case subPath == "/new-order":
		s.handleNewOrder(w, r, caName)
	case strings.HasPrefix(subPath, "/authz/"):
		authzID := strings.TrimPrefix(subPath, "/authz/")
		s.handleAuthz(w, r, caName, authzID)
	case strings.HasPrefix(subPath, "/challenge/"):
		challengeID := strings.TrimPrefix(subPath, "/challenge/")
		s.handleChallenge(w, r, caName, challengeID)
	case strings.HasPrefix(subPath, "/order/") && strings.HasSuffix(subPath, "/finalize"):
		orderID := strings.TrimPrefix(subPath, "/order/")
		orderID = strings.TrimSuffix(orderID, "/finalize")
		s.handleFinalize(w, r, caName, orderID)
	case strings.HasPrefix(subPath, "/order/"):
		orderID := strings.TrimPrefix(subPath, "/order/")
		s.handleGetOrder(w, r, caName, orderID)
	case strings.HasPrefix(subPath, "/certificate/"):
		certID := strings.TrimPrefix(subPath, "/certificate/")
		s.handleCertificate(w, r, caName, certID)
	default:
		http.NotFound(w, r)
	}
}

// --- Handlers ---

func (s *Server) handleDirectory(w http.ResponseWriter, _ *http.Request, caName string) {
	base := s.caURL(caName)
	dir := map[string]interface{}{
		"newNonce":   base + "/new-nonce",
		"newAccount": base + "/new-account",
		"newOrder":   base + "/new-order",
		"revokeCert": base + "/revoke-cert",
		"keyChange":  base + "/key-change",
	}
	s.writeJSON(w, http.StatusOK, dir)
}

func (s *Server) handleNewNonce(w http.ResponseWriter, r *http.Request) {
	// Nonce is already set in ServeHTTP
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == "HEAD" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleNewAccount(w http.ResponseWriter, r *http.Request, caName string) {
	// POC: skip JWS validation, parse payload as JSON
	payload, err := s.readPayload(r)
	if err != nil {
		s.writeACMEError(w, http.StatusBadRequest, "urn:ietf:params:acme:error:malformed", err.Error())
		return
	}

	var req struct {
		Contact []string `json:"contact"`
	}
	if len(payload) > 0 {
		json.Unmarshal(payload, &req)
	}

	acct := s.store.CreateAccount(req.Contact)

	w.Header().Set("Location", s.caURL(caName)+"/account/"+acct.ID)
	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  acct.Status,
		"contact": acct.Contact,
		"orders":  s.caURL(caName) + "/orders",
	})
}

func (s *Server) handleNewOrder(w http.ResponseWriter, r *http.Request, caName string) {
	payload, err := s.readPayload(r)
	if err != nil {
		s.writeACMEError(w, http.StatusBadRequest, "urn:ietf:params:acme:error:malformed", err.Error())
		return
	}

	var req struct {
		Identifiers []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"identifiers"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		s.writeACMEError(w, http.StatusBadRequest, "urn:ietf:params:acme:error:malformed",
			"invalid order request")
		return
	}

	var identifiers []api.Identifier
	for _, id := range req.Identifiers {
		identifiers = append(identifiers, api.Identifier{Type: id.Type, Value: id.Value})
	}

	order := s.store.CreateOrder(caName, identifiers)

	w.Header().Set("Location", s.orderURL(caName, order.ID))
	s.writeOrderJSON(w, http.StatusCreated, order, caName)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, _ *http.Request, caName, orderID string) {
	order, ok := s.store.GetOrder(orderID)
	if !ok {
		s.writeACMEError(w, http.StatusNotFound, "urn:ietf:params:acme:error:malformed", "order not found")
		return
	}
	s.writeOrderJSON(w, http.StatusOK, order, caName)
}

func (s *Server) handleAuthz(w http.ResponseWriter, _ *http.Request, caName, authzID string) {
	authz, ok := s.store.GetAuthorization(authzID)
	if !ok {
		s.writeACMEError(w, http.StatusNotFound, "urn:ietf:params:acme:error:malformed", "authorization not found")
		return
	}

	challenges := make([]map[string]interface{}, len(authz.Challenges))
	for i, ch := range authz.Challenges {
		challenges[i] = map[string]interface{}{
			"type":   ch.Type,
			"url":    s.caURL(caName) + "/challenge/" + ch.ID,
			"token":  ch.Token,
			"status": ch.Status,
		}
	}

	resp := map[string]interface{}{
		"status": authz.Status,
		"identifier": map[string]string{
			"type":  authz.Identifier.Type,
			"value": authz.Identifier.Value,
		},
		"challenges": challenges,
		"expires":    authz.ExpiresAt.Format(time.RFC3339),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChallenge(w http.ResponseWriter, r *http.Request, caName, challengeID string) {
	ch, ok := s.store.GetChallenge(challengeID)
	if !ok {
		s.writeACMEError(w, http.StatusNotFound, "urn:ietf:params:acme:error:malformed", "challenge not found")
		return
	}

	// If POST with payload, this is a challenge response from the device
	if r.Method == "POST" {
		payload, _ := s.readPayload(r)
		ch.Payload = payload

		issuer := s.issuers[caName]

		// Find the parent order for this challenge
		authz, _ := s.store.GetAuthorization(ch.AuthzID)
		var order *api.Order
		if authz != nil {
			order, _ = s.store.GetOrder(authz.OrderID)
		}

		// Delegate challenge validation to the backend
		if err := issuer.ValidateChallenge(r.Context(), ch, order); err != nil {
			s.store.UpdateChallengeStatus(ch.ID, "invalid")
			s.store.UpdateAuthzStatus(ch.AuthzID, "invalid")
			s.writeACMEError(w, http.StatusForbidden, "urn:ietf:params:acme:error:unauthorized", err.Error())
			return
		}

		// Validation passed
		s.store.UpdateChallengeStatus(ch.ID, "valid")
		s.store.UpdateAuthzStatus(ch.AuthzID, "valid")

		// Check if order is now ready
		if order != nil {
			s.store.CheckOrderReady(order.ID)
		}
	}

	// Reload challenge after potential updates
	ch, _ = s.store.GetChallenge(challengeID)

	resp := map[string]interface{}{
		"type":   ch.Type,
		"url":    s.caURL(caName) + "/challenge/" + ch.ID,
		"token":  ch.Token,
		"status": ch.Status,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleFinalize(w http.ResponseWriter, r *http.Request, caName, orderID string) {
	order, ok := s.store.GetOrder(orderID)
	if !ok {
		s.writeACMEError(w, http.StatusNotFound, "urn:ietf:params:acme:error:malformed", "order not found")
		return
	}

	if order.Status != "ready" {
		s.writeACMEError(w, http.StatusForbidden, "urn:ietf:params:acme:error:orderNotReady",
			fmt.Sprintf("order status is %q, expected \"ready\"", order.Status))
		return
	}

	payload, err := s.readPayload(r)
	if err != nil {
		s.writeACMEError(w, http.StatusBadRequest, "urn:ietf:params:acme:error:malformed", err.Error())
		return
	}

	var req struct {
		CSR string `json:"csr"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		s.writeACMEError(w, http.StatusBadRequest, "urn:ietf:params:acme:error:malformed", "invalid finalize request")
		return
	}

	csrDER, err := base64.RawURLEncoding.DecodeString(req.CSR)
	if err != nil {
		s.writeACMEError(w, http.StatusBadRequest, "urn:ietf:params:acme:error:malformed", "invalid CSR encoding")
		return
	}

	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		s.writeACMEError(w, http.StatusBadRequest, "urn:ietf:params:acme:error:badCSR", "invalid CSR")
		return
	}

	// Transition to processing
	s.store.UpdateOrderStatus(orderID, "processing")
	s.store.SetOrderCSR(orderID, csrDER)

	// Delegate certificate issuance to the backend
	issuer := s.issuers[caName]
	issued, err := issuer.IssueCertificate(r.Context(), csr, order)
	if err != nil {
		s.store.UpdateOrderStatus(orderID, "invalid")
		s.logger.Error("certificate issuance failed", "err", err, "ca", caName, "order", orderID)
		s.writeACMEError(w, http.StatusInternalServerError, "urn:ietf:params:acme:error:serverInternal",
			"certificate issuance failed")
		return
	}

	// Store the certificate as PEM
	certPEM := derChainToPEM(issued.DERChain)
	certID := s.store.StoreCertificate(certPEM)
	s.store.SetOrderCertID(orderID, certID)

	s.logger.Info("certificate issued",
		"ca", caName,
		"order", orderID,
		"cert", certID,
		"subject", csr.Subject.CommonName,
	)

	// Return the updated order
	order, _ = s.store.GetOrder(orderID)
	s.writeOrderJSON(w, http.StatusOK, order, caName)
}

func (s *Server) handleCertificate(w http.ResponseWriter, _ *http.Request, _, certID string) {
	certPEM, ok := s.store.GetCertificate(certID)
	if !ok {
		s.writeACMEError(w, http.StatusNotFound, "urn:ietf:params:acme:error:malformed", "certificate not found")
		return
	}

	w.Header().Set("Content-Type", "application/pem-certificate-chain")
	w.WriteHeader(http.StatusOK)
	w.Write(certPEM)
}

// --- Response helpers ---

func (s *Server) caURL(caName string) string {
	return s.baseURL + PathPrefix + caName
}

func (s *Server) orderURL(caName, orderID string) string {
	return s.caURL(caName) + "/order/" + orderID
}

func (s *Server) writeOrderJSON(w http.ResponseWriter, status int, order *api.Order, caName string) {
	authzURLs := make([]string, len(order.Authorizations))
	for i, id := range order.Authorizations {
		authzURLs[i] = s.caURL(caName) + "/authz/" + id
	}

	ids := make([]map[string]string, len(order.Identifiers))
	for i, id := range order.Identifiers {
		ids[i] = map[string]string{"type": id.Type, "value": id.Value}
	}

	resp := map[string]interface{}{
		"status":         order.Status,
		"identifiers":    ids,
		"authorizations": authzURLs,
		"finalize":       s.orderURL(caName, order.ID) + "/finalize",
		"expires":        order.ExpiresAt.Format(time.RFC3339),
	}
	if order.CertID != "" {
		resp["certificate"] = s.caURL(caName) + "/certificate/" + order.CertID
	}

	w.Header().Set("Location", s.orderURL(caName, order.ID))
	s.writeJSON(w, status, resp)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) writeACMEError(w http.ResponseWriter, status int, errType, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":   errType,
		"detail": detail,
		"status": status,
	})
}

// readPayload reads the request body. For the POC, we accept both plain JSON
// and JWS-wrapped payloads (extracting the payload from JWS if detected).
func (s *Server) readPayload(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	if len(body) == 0 {
		return []byte("{}"), nil
	}

	// Check if this is a JWS (has "protected", "payload", "signature" fields)
	var jws struct {
		Protected string `json:"protected"`
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}
	if json.Unmarshal(body, &jws) == nil && jws.Protected != "" && jws.Signature != "" {
		// Extract the payload from the JWS
		if jws.Payload == "" {
			return []byte("{}"), nil // POST-as-GET
		}
		decoded, err := base64.RawURLEncoding.DecodeString(jws.Payload)
		if err != nil {
			return nil, fmt.Errorf("decoding JWS payload: %w", err)
		}
		return decoded, nil
	}

	// Plain JSON
	return body, nil
}

func derChainToPEM(derChain [][]byte) []byte {
	var pemData []byte
	for _, der := range derChain {
		block := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		}
		pemData = append(pemData, pem.EncodeToMemory(block)...)
	}
	return pemData
}
