package servicediscovery

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/env"
	runnerv1 "github.com/kofuk/premises/backend/runner/gen/runner/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func Run(ctx context.Context, config *runner.Config, args []string) int {
	socketPath := env.DataPath("service-discovery.sock")
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to listen on service discovery socket", slog.Any("error", err))
		return 1
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	health := health.NewServer()
	serviceDiscovery := &ServiceDiscoveryServer{}

	runnerv1.RegisterServiceDiscoveryServiceServer(grpcServer, serviceDiscovery)
	grpc_health_v1.RegisterHealthServer(grpcServer, health)
	reflection.Register(grpcServer)

	health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "Shutting down service discovery gRPC server")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		slog.ErrorContext(ctx, "Failed to serve service discovery gRPC server", slog.Any("error", err))

		return 1
	}

	return 0
}
