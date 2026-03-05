//go:build !windows

package enforcement

import "context"

// RegistryHandler is a stub for non-Windows platforms.
type RegistryHandler struct{}

func NewRegistryHandler() *RegistryHandler { return &RegistryHandler{} }
func (h *RegistryHandler) Name() string    { return "registry" }
func (h *RegistryHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	return nil, ErrNotSupported
}
func (h *RegistryHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	return nil, ErrNotSupported
}
