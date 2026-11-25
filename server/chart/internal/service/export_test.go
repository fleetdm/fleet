package service

import (
	"context"
	"testing"
	"time"
)

// Exported for the tests in package service_test, which use the MySQL store.

type AllowAllAuthorizer = mockAuthorizer

func GlobalViewer() *mockViewerProvider { return globalViewer() }

func PremiumCtx(t *testing.T) context.Context { return premiumCtx(t) }

func SetNow(s *Service, now func() time.Time) { s.now = now }
