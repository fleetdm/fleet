package scepserver

import (
	"context"
	"log/slog"
	"time"
)

type loggingService struct {
	logger *slog.Logger
	Service
}

// NewLoggingService creates adds logging to the SCEP service
func NewLoggingService(logger *slog.Logger, s Service) Service {
	return &loggingService{logger, s}
}

func (mw *loggingService) GetCACaps(ctx context.Context) (caps []byte, err error) {
	defer func(begin time.Time) {
		mw.logger.InfoContext(ctx, "GetCACaps",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	caps, err = mw.Service.GetCACaps(ctx)
	return
}

func (mw *loggingService) GetCACert(ctx context.Context, message string) (cert []byte, certNum int, err error) {
	defer func(begin time.Time) {
		mw.logger.InfoContext(ctx, "GetCACert",
			"message", message,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	cert, certNum, err = mw.Service.GetCACert(ctx, message)
	return
}

func (mw *loggingService) PKIOperation(ctx context.Context, data []byte) (certRep []byte, err error) {
	defer func(begin time.Time) {
		mw.logger.InfoContext(ctx, "PKIOperation",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	certRep, err = mw.Service.PKIOperation(ctx, data)
	return
}
