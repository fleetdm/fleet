//go:build !windows

package enforcement

import "context"

// AuditHandler is a stub for non-Windows platforms.
type AuditHandler struct{}

func NewAuditHandler() *AuditHandler     { return &AuditHandler{} }
func (h *AuditHandler) Name() string     { return "audit" }
func (h *AuditHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	return nil, ErrNotSupported
}
func (h *AuditHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	return nil, ErrNotSupported
}
