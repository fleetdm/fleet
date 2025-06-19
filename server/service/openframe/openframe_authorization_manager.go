package openframe

import (
	"sync"
)

// OpenFrameAuthorizationManager manages OpenFrame authentication tokens
type OpenFrameAuthorizationManager struct {
	token string
	mu    sync.RWMutex
}

// NewOpenFrameAuthorizationManager creates a new OpenFrameAuthorizationManager instance
func NewOpenFrameAuthorizationManager() *OpenFrameAuthorizationManager {
	return &OpenFrameAuthorizationManager{}
}

// NewOpenFrameAuthorizationManagerWithToken creates a new OpenFrameAuthorizationManager instance with an initial token
func NewOpenFrameAuthorizationManagerWithToken(token string) *OpenFrameAuthorizationManager {
	return &OpenFrameAuthorizationManager{
		token: token,
	}
}

// UpdateToken updates the stored authentication token
func (m *OpenFrameAuthorizationManager) UpdateToken(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = token
}

// GetToken returns the current authentication token
func (m *OpenFrameAuthorizationManager) GetToken() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.token
} 