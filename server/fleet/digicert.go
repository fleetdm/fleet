package fleet

import (
	"context"
	"time"
)

type DigiCertCertificate struct {
	PfxData       []byte
	Password      string
	NotValidAfter time.Time
}

type DigiCertService interface {
	VerifyProfileID(ctx context.Context, config DigiCertIntegration) error
	GetCertificate(ctx context.Context, config DigiCertIntegration) (*DigiCertCertificate, error)
}
