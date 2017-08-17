package launcher

import (
	pb "github.com/kolide/agent-api"
	"github.com/kolide/fleet/server/kolide"

	"golang.org/x/net/context"
)

type authMiddleware struct {
	svc  kolide.OsqueryService
	next pb.ApiServer
}

func newAuthMiddleware(svc kolide.OsqueryService) func(svc pb.ApiServer) pb.ApiServer {
	return func(next pb.ApiServer) pb.ApiServer {
		return authMiddleware{
			svc:  svc,
			next: next,
		}
	}
}

func (s authMiddleware) RequestEnrollment(ctx context.Context, req *pb.EnrollmentRequest) (*pb.EnrollmentResponse, error) {
	return s.next.RequestEnrollment(ctx, req)
}

func (s authMiddleware) RequestConfig(ctx context.Context, req *pb.AgentApiRequest) (*pb.ConfigResponse, error) {
	authCtx, auth, err := s.authenticateHost(ctx, req.NodeKey)
	if err != nil {
		return nil, err
	}
	if auth.nodeInvalid {
		return &pb.ConfigResponse{NodeInvalid: auth.nodeInvalid, ErrorCode: auth.errorCode}, nil
	}
	return s.next.RequestConfig(authCtx, req)
}

func (s authMiddleware) RequestQueries(ctx context.Context, req *pb.AgentApiRequest) (resp *pb.QueryCollection, err error) {
	authCtx, auth, err := s.authenticateHost(ctx, req.NodeKey)
	if err != nil {
		return nil, err
	}
	if auth.nodeInvalid {
		return &pb.QueryCollection{NodeInvalid: auth.nodeInvalid, ErrorCode: auth.errorCode}, nil
	}
	return s.next.RequestQueries(authCtx, req)
}

func (s authMiddleware) PublishLogs(ctx context.Context, req *pb.LogCollection) (resp *pb.AgentApiResponse, err error) {
	authCtx, auth, err := s.authenticateHost(ctx, req.NodeKey)
	if err != nil {
		return nil, err
	}
	if auth.nodeInvalid {
		return &pb.AgentApiResponse{NodeInvalid: auth.nodeInvalid, ErrorCode: auth.errorCode}, nil
	}
	return s.next.PublishLogs(authCtx, req)
}

func (s authMiddleware) PublishResults(ctx context.Context, req *pb.ResultCollection) (resp *pb.AgentApiResponse, err error) {
	authCtx, auth, err := s.authenticateHost(ctx, req.NodeKey)
	if err != nil {
		return nil, err
	}
	if auth.nodeInvalid {
		return &pb.AgentApiResponse{NodeInvalid: auth.nodeInvalid, ErrorCode: auth.errorCode}, nil
	}
	return s.next.PublishResults(authCtx, req)
}

func (s authMiddleware) HotConfigure(req *pb.AgentApiRequest, svr pb.Api_HotConfigureServer) (err error) {
	if _, _, err := s.authenticateHost(context.Background(), req.NodeKey); err != nil {
		return err
	}
	return s.next.HotConfigure(req, svr)
}

func (s authMiddleware) HotlineBling(svr pb.Api_HotlineBlingServer) (err error) {
	return s.next.HotlineBling(svr)
}

type auth struct {
	nodeInvalid bool
	errorCode   string
	host        *kolide.Host
}

func (s authMiddleware) authenticateHost(ctx context.Context, nodeKey string) (context.Context, *auth, error) {
	host, err := s.svc.AuthenticateHost(newCtx(ctx), nodeKey)
	if err != nil {
		if errEnroll, ok := err.(enrollmentError); ok {
			return ctx, &auth{nodeInvalid: errEnroll.NodeInvalid(), errorCode: errEnroll.Error()}, nil
		}
		return nil, nil, err
	}
	return withHost(ctx, *host), &auth{host: host}, nil
}
