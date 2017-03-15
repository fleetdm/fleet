package service

import (
	"context"
	"time"

	"github.com/kolide/kolide/server/kolide"
)

func (mw loggingMiddleware) EnrollAgent(ctx context.Context, enrollSecret string, hostIdentifier string) (string, error) {
	var (
		nodeKey string
		err     error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "EnrollAgent",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	nodeKey, err = mw.Service.EnrollAgent(ctx, enrollSecret, hostIdentifier)
	return nodeKey, err
}

func (mw loggingMiddleware) AuthenticateHost(ctx context.Context, nodeKey string) (*kolide.Host, error) {
	var (
		host *kolide.Host
		err  error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "AuthenticateHost",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	host, err = mw.Service.AuthenticateHost(ctx, nodeKey)
	return host, err
}

func (mw loggingMiddleware) GetClientConfig(ctx context.Context) (*kolide.OsqueryConfig, error) {
	var (
		config *kolide.OsqueryConfig
		err    error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "GetClientConfig",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	config, err = mw.Service.GetClientConfig(ctx)
	return config, err
}

func (mw loggingMiddleware) GetDistributedQueries(ctx context.Context) (map[string]string, error) {
	var (
		queries map[string]string
		err     error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "GetDistributedQueries",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	queries, err = mw.Service.GetDistributedQueries(ctx)
	return queries, err
}

func (mw loggingMiddleware) SubmitDistributedQueryResults(ctx context.Context, results kolide.OsqueryDistributedQueryResults, statuses map[string]string) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "SubmitDistributedQueryResults",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.SubmitDistributedQueryResults(ctx, results, statuses)
	return err
}

func (mw loggingMiddleware) SubmitStatusLogs(ctx context.Context, logs []kolide.OsqueryStatusLog) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "SubmitStatusLogs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.SubmitStatusLogs(ctx, logs)
	return err
}

func (mw loggingMiddleware) SubmitResultLogs(ctx context.Context, logs []kolide.OsqueryResultLog) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "SubmitResultLogs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.SubmitResultLogs(ctx, logs)
	return err
}
