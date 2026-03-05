//go:build !windows

package enforcement

import "context"

// ServiceHandler is a stub for non-Windows platforms.
type ServiceHandler struct{}

func NewServiceHandler() *ServiceHandler     { return &ServiceHandler{} }
func (h *ServiceHandler) Name() string       { return "service" }
func (h *ServiceHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	return nil, ErrNotSupported
}
func (h *ServiceHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	return nil, ErrNotSupported
}
