// Package multi contains a multi-service dispatcher.
package multi

import (
	"context"
	"sync"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// MultiService executes multiple services for the same service calls.
// The first service returns values or errors to the caller. We give the
// first service a chance to alter any 'core' request data (say, the
// Enrollment ID) by waiting for it to finish then we run the remaining
// services' calls in parallel.
type MultiService struct {
	logger log.Logger
	svcs   []service.CheckinAndCommandService
	ctx    context.Context
}

func New(logger log.Logger, svcs ...service.CheckinAndCommandService) *MultiService {
	if len(svcs) < 1 {
		panic("must supply at least one service")
	}
	return &MultiService{
		logger: logger,
		svcs:   svcs,
		ctx:    context.Background(),
	}
}

type errorRunner func(service.CheckinAndCommandService) error

func (ms *MultiService) runOthers(ctx context.Context, r errorRunner) {
	var wg sync.WaitGroup

	for i, svc := range ms.svcs[1:] {
		wg.Add(1)
		go func(n int, s service.CheckinAndCommandService) {
			defer wg.Done()

			err := r(s)
			if err != nil {
				ctxlog.Logger(ctx, ms.logger).Info(
					"sub_service", n,
					"err", err,
				)
			}
		}(i+1, svc)
	}
	wg.Wait()
}

// RequestWithContext returns a clone of r and sets its context to ctx.
func (ms *MultiService) RequestWithContext(r *mdm.Request) *mdm.Request {
	r2 := r.Clone()
	r2.Context = ms.ctx
	return r2
}

func (ms *MultiService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	err := ms.svcs[0].Authenticate(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		return svc.Authenticate(rc, m)
	})
	return err
}

func (ms *MultiService) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	err := ms.svcs[0].TokenUpdate(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		return svc.TokenUpdate(rc, m)
	})
	return err
}

func (ms *MultiService) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	err := ms.svcs[0].CheckOut(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		return svc.CheckOut(rc, m)
	})
	return err
}

func (ms *MultiService) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	respBytes, err := ms.svcs[0].UserAuthenticate(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		_, err := svc.UserAuthenticate(rc, m)
		return err
	})
	return respBytes, err
}

func (ms *MultiService) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	err := ms.svcs[0].SetBootstrapToken(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		return svc.SetBootstrapToken(rc, m)
	})
	return err
}

func (ms *MultiService) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	bsToken, err := ms.svcs[0].GetBootstrapToken(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		_, err := svc.GetBootstrapToken(rc, m)
		return err
	})
	return bsToken, err
}

func (ms *MultiService) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	retBytes, err := ms.svcs[0].DeclarativeManagement(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		_, err := svc.DeclarativeManagement(rc, m)
		return err
	})
	return retBytes, err
}

func (ms *MultiService) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	resp, err := ms.svcs[0].GetToken(r, m)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		_, err := svc.GetToken(rc, m)
		return err
	})
	return resp, err
}

func (ms *MultiService) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	cmd, err := ms.svcs[0].CommandAndReportResults(r, results)
	rc := ms.RequestWithContext(r)
	ms.runOthers(r.Context, func(svc service.CheckinAndCommandService) error {
		_, err := svc.CommandAndReportResults(rc, results)
		return err
	})
	return cmd, err
}
