package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/acme/api"
)

// Store manages ACME server state (accounts, orders, authorizations, challenges,
// nonces, certificates). This is an in-memory implementation for the POC;
// a MySQL implementation can replace it for production.
type Store struct {
	mu sync.RWMutex

	accounts       map[string]*Account           // accountID -> Account
	orders         map[string]*api.Order         // orderID -> Order
	authorizations map[string]*api.Authorization // authzID -> Authorization
	challenges     map[string]*api.Challenge     // challengeID -> Challenge
	certificates   map[string][]byte             // certID -> PEM chain
	nonces         map[string]time.Time          // nonce -> expiry
}

// Account represents an ACME account managed by the server.
type Account struct {
	ID      string
	Status  string
	Contact []string
}

// NewStore creates a new in-memory store.
func NewStore() *Store {
	return &Store{
		accounts:       make(map[string]*Account),
		orders:         make(map[string]*api.Order),
		authorizations: make(map[string]*api.Authorization),
		challenges:     make(map[string]*api.Challenge),
		certificates:   make(map[string][]byte),
		nonces:         make(map[string]time.Time),
	}
}

// --- Nonces ---

// CreateNonce generates and stores a new nonce.
func (s *Store) CreateNonce() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	nonce := randomID(16)
	s.nonces[nonce] = time.Now().Add(1 * time.Hour)
	return nonce
}

// ConsumeNonce validates and removes a nonce. Returns false if invalid/expired.
func (s *Store) ConsumeNonce(nonce string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiry, ok := s.nonces[nonce]
	if !ok {
		return false
	}
	delete(s.nonces, nonce)
	return time.Now().Before(expiry)
}

// --- Accounts ---

// CreateAccount creates a new account and returns it.
func (s *Store) CreateAccount(contact []string) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()

	acct := &Account{
		ID:      randomID(12),
		Status:  "valid",
		Contact: contact,
	}
	s.accounts[acct.ID] = acct
	return acct
}

// GetAccount returns an account by ID.
func (s *Store) GetAccount(id string) (*Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acct, ok := s.accounts[id]
	return acct, ok
}

// --- Orders ---

// CreateOrder creates a new order with authorizations and challenges.
func (s *Store) CreateOrder(caName string, identifiers []api.Identifier) *api.Order {
	s.mu.Lock()
	defer s.mu.Unlock()

	order := &api.Order{
		ID:          randomID(12),
		CAName:      caName,
		Status:      "pending",
		Identifiers: identifiers,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
	}

	// Create an authorization and challenge for each identifier
	for _, id := range identifiers {
		challenge := &api.Challenge{
			ID:     randomID(12),
			Type:   "device-attest-01",
			Token:  randomID(16),
			Status: "pending",
		}

		authz := &api.Authorization{
			ID:         randomID(12),
			OrderID:    order.ID,
			Identifier: id,
			Status:     "pending",
			Challenges: []api.Challenge{*challenge},
			ExpiresAt:  order.ExpiresAt,
		}

		challenge.AuthzID = authz.ID
		s.challenges[challenge.ID] = challenge
		s.authorizations[authz.ID] = authz
		order.Authorizations = append(order.Authorizations, authz.ID)
	}

	s.orders[order.ID] = order
	return order
}

// GetOrder returns an order by ID.
func (s *Store) GetOrder(id string) (*api.Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	return order, ok
}

// UpdateOrderStatus updates the status of an order.
func (s *Store) UpdateOrderStatus(id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[id]
	if !ok {
		return fmt.Errorf("order not found: %s", id)
	}
	order.Status = status
	return nil
}

// SetOrderCSR stores the CSR on the order.
func (s *Store) SetOrderCSR(id string, csr []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[id]
	if !ok {
		return fmt.Errorf("order not found: %s", id)
	}
	order.CSR = csr
	return nil
}

// SetOrderCertID stores the certificate ID on a completed order.
func (s *Store) SetOrderCertID(id, certID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[id]
	if !ok {
		return fmt.Errorf("order not found: %s", id)
	}
	order.CertID = certID
	order.Status = "valid"
	return nil
}

// --- Authorizations ---

// GetAuthorization returns an authorization by ID.
func (s *Store) GetAuthorization(id string) (*api.Authorization, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	authz, ok := s.authorizations[id]
	return authz, ok
}

// UpdateAuthzStatus updates the status of an authorization.
func (s *Store) UpdateAuthzStatus(id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	authz, ok := s.authorizations[id]
	if !ok {
		return fmt.Errorf("authorization not found: %s", id)
	}
	authz.Status = status
	// Also update the challenge statuses embedded in the authz
	for i := range authz.Challenges {
		if authz.Challenges[i].Status == "pending" || authz.Challenges[i].Status == "processing" {
			authz.Challenges[i].Status = status
		}
	}
	return nil
}

// --- Challenges ---

// GetChallenge returns a challenge by ID.
func (s *Store) GetChallenge(id string) (*api.Challenge, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.challenges[id]
	return ch, ok
}

// UpdateChallengeStatus updates the status of a challenge and its parent authz.
func (s *Store) UpdateChallengeStatus(id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.challenges[id]
	if !ok {
		return fmt.Errorf("challenge not found: %s", id)
	}
	ch.Status = status

	// Also update the embedded challenge in the authorization
	if authz, ok := s.authorizations[ch.AuthzID]; ok {
		for i := range authz.Challenges {
			if authz.Challenges[i].ID == id {
				authz.Challenges[i].Status = status
				break
			}
		}
	}
	return nil
}

// --- Certificates ---

// StoreCertificate stores a PEM-encoded certificate chain.
func (s *Store) StoreCertificate(certPEM []byte) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := randomID(12)
	s.certificates[id] = certPEM
	return id
}

// GetCertificate returns a PEM-encoded certificate chain by ID.
func (s *Store) GetCertificate(id string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cert, ok := s.certificates[id]
	return cert, ok
}

// --- Helpers ---

// CheckOrderReady checks if all authorizations for an order are valid,
// and if so, transitions the order to "ready" status.
func (s *Store) CheckOrderReady(orderID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok || order.Status != "pending" {
		return
	}

	allValid := true
	for _, authzID := range order.Authorizations {
		authz, ok := s.authorizations[authzID]
		if !ok || authz.Status != "valid" {
			allValid = false
			break
		}
	}

	if allValid {
		order.Status = "ready"
	}
}

func randomID(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand.Read failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
