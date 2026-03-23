package servicediscovery

import (
	runnerv1 "github.com/kofuk/premises/backend/runner/gen/runner/v1"
)

type ServiceDiscoveryServer struct {
	runnerv1.UnimplementedServiceDiscoveryServiceServer
}
