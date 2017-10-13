package launcher

import (
	"time"

	kitlog "github.com/go-kit/kit/log"
	pb "github.com/kolide/agent-api"
	"golang.org/x/net/context"
)

type loggingMiddleware struct {
	logger kitlog.Logger
	next   pb.ApiServer
}

func newLoggingMiddleware(logger kitlog.Logger) func(svc pb.ApiServer) pb.ApiServer {
	return func(next pb.ApiServer) pb.ApiServer {
		return loggingMiddleware{
			logger: kitlog.With(logger, "component", "gRPC Launcher"),
			next:   next,
		}
	}
}

func (s loggingMiddleware) RequestEnrollment(ctx context.Context, req *pb.EnrollmentRequest) (resp *pb.EnrollmentResponse, err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "RequestEnrollment",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return s.next.RequestEnrollment(ctx, req)
}

func (s loggingMiddleware) RequestConfig(ctx context.Context, req *pb.AgentApiRequest) (resp *pb.ConfigResponse, err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "RequestConfig",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return s.next.RequestConfig(ctx, req)
}

func (s loggingMiddleware) RequestQueries(ctx context.Context, req *pb.AgentApiRequest) (resp *pb.QueryCollection, err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "RequestQueries",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return s.next.RequestQueries(ctx, req)
}

func (s loggingMiddleware) PublishLogs(ctx context.Context, req *pb.LogCollection) (resp *pb.AgentApiResponse, err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "PublishLogs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return s.next.PublishLogs(ctx, req)
}

func (s loggingMiddleware) PublishResults(ctx context.Context, req *pb.ResultCollection) (resp *pb.AgentApiResponse, err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "PublishResults",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return s.next.PublishResults(ctx, req)
}

func (s loggingMiddleware) CheckHealth(ctx context.Context, coll *pb.AgentApiRequest) (resp *pb.HealthCheckResponse, err error) {
	defer func(begin time.Time) {
		s.logger.Log(
			"method", "CheckHealth",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return s.next.CheckHealth(ctx, coll)
}
