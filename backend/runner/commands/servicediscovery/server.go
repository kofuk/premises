package servicediscovery

import (
	"context"

	"github.com/kofuk/premises/backend/runner/commands/servicediscovery/internal"
	runnerv1 "github.com/kofuk/premises/backend/runner/gen/runner/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServiceDiscoveryServer struct {
	runnerv1.UnimplementedServiceDiscoveryServiceServer
	repository *internal.Repository
}

func NewServiceDiscoveryServer(repository *internal.Repository) *ServiceDiscoveryServer {
	return &ServiceDiscoveryServer{
		repository: repository,
	}
}

func (s *ServiceDiscoveryServer) InternalReset(ctx context.Context, req *runnerv1.ResetServiceDiscoveryRequest) (*runnerv1.ResetServiceDiscoveryResponse, error) {
	if err := s.repository.Reset(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reset service discovery repository: %v", err)
	}

	return &runnerv1.ResetServiceDiscoveryResponse{}, nil
}

func (s *ServiceDiscoveryServer) Expose(ctx context.Context, req *runnerv1.ExposeServiceRequest) (*runnerv1.ExposeServiceResponse, error) {
	if req.Service == nil {
		return nil, status.Errorf(codes.InvalidArgument, "service is required")
	}
	if req.Kind == runnerv1.ServiceKind_KIND_UNKNOWN {
		return nil, status.Errorf(codes.InvalidArgument, "service kind is required")
	}

	if err := s.repository.AddOrUpdateService(req.Kind, req.Service); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add or update service: %v", err)
	}

	return &runnerv1.ExposeServiceResponse{}, nil
}

func (s *ServiceDiscoveryServer) Resolve(ctx context.Context, req *runnerv1.ResolveServiceRequest) (*runnerv1.ResolveServiceResponse, error) {
	if req.Kind == runnerv1.ServiceKind_KIND_UNKNOWN {
		return nil, status.Errorf(codes.InvalidArgument, "service is required")
	}

	service, err := s.repository.GetServiceByKind(req.Kind)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "service of kind %v not found", req.Kind.String())
	}

	return &runnerv1.ResolveServiceResponse{
		Service: service,
	}, nil
}
