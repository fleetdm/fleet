//go:build !windows

package enforcement

import "context"

// SecpolHandler is a stub for non-Windows platforms.
type SecpolHandler struct{}

func NewSecpolHandler() *SecpolHandler     { return &SecpolHandler{} }
func (h *SecpolHandler) Name() string      { return "secpol" }
func (h *SecpolHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	return nil, ErrNotSupported
}
func (h *SecpolHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	return nil, ErrNotSupported
}
