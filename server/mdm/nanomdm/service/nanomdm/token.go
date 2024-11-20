package nanomdm

import (
	"errors"
	"fmt"
	"sync"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
)

// StaticToken holds static token bytes.
type StaticToken struct {
	token []byte
}

// NewStaticToken creates a new static token handler.
func NewStaticToken(token []byte) *StaticToken {
	return &StaticToken{token: token}
}

// GetToken always responds with the static token bytes.
func (t *StaticToken) GetToken(_ *mdm.Request, _ *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	return &mdm.GetTokenResponse{TokenData: t.token}, nil
}

// TokenMux is a middleware multiplexer for GetToken check-in messages.
// A TokenServiceType string is associated with a GetToken handler and
// then dispatched appropriately.
type TokenMux struct {
	typesMu sync.RWMutex
	types   map[string]service.GetToken
}

// NewTokenMux creates a new TokenMux.
func NewTokenMux() *TokenMux { return &TokenMux{} }

// Handle registers a GetToken handler for the given service type.
// See https://developer.apple.com/documentation/devicemanagement/gettokenrequest
func (mux *TokenMux) Handle(serviceType string, handler service.GetToken) {
	if serviceType == "" {
		panic("tokenmux: invalid service type")
	}
	if handler == nil {
		panic("tokenmux: invalid handler")
	}
	mux.typesMu.Lock()
	defer mux.typesMu.Unlock()
	if mux.types == nil {
		mux.types = make(map[string]service.GetToken)
	} else if _, exists := mux.types[serviceType]; exists {
		panic("tokenmux: multiple registrations for " + serviceType)
	}
	mux.types[serviceType] = handler
}

// GetToken is the middleware that dispatches a GetToken handler based on service type.
func (mux *TokenMux) GetToken(r *mdm.Request, t *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	if t == nil {
		return nil, errors.New("nil MDM GetToken")
	}
	var next service.GetToken
	mux.typesMu.RLock()
	if mux.types != nil {
		next = mux.types[t.TokenServiceType]
	}
	mux.typesMu.RUnlock()
	if next == nil {
		return nil, fmt.Errorf("no handler for TokenServiceType: %v", t.TokenServiceType)
	}
	return next.GetToken(r, t)
}
