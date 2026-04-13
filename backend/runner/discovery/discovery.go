package discovery

import (
	"sync"

	"github.com/kofuk/premises/backend/runner/env"
	runnerv1 "github.com/kofuk/premises/backend/runner/gen/runner/v1"
	"google.golang.org/grpc"
)

var (
	client     runnerv1.ServiceDiscoveryServiceClient
	clientOnce sync.Once
)

func GetServiceDiscoveryClient() (runnerv1.ServiceDiscoveryServiceClient, error) {
	var initErr error

	clientOnce.Do(func() {
		conn, err := grpc.NewClient("unix://" + env.DataPath("service-discovery.sock"))
		if err != nil {
			initErr = err
			return
		}

		client = runnerv1.NewServiceDiscoveryServiceClient(conn)
	})

	return client, initErr
}
