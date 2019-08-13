package service

import "context"

func (svc service) StatusResultStore(ctx context.Context) error {
	return svc.resultStore.HealthCheck()
}
