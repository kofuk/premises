package internal

import (
	"sync"

	runnerv1 "github.com/kofuk/premises/backend/runner/gen/runner/v1"
)

type Repository struct {
	services map[runnerv1.ServiceKind]*runnerv1.ExposedService
	mu       sync.Mutex
}

func NewRepository() *Repository {
	return &Repository{
		services: make(map[runnerv1.ServiceKind]*runnerv1.ExposedService),
	}
}

func (r *Repository) Initialize() error {
	return nil
}

func (r *Repository) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = make(map[runnerv1.ServiceKind]*runnerv1.ExposedService)

	return nil
}

func (r *Repository) AddOrUpdateService(kind runnerv1.ServiceKind, service *runnerv1.ExposedService) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services[kind] = service

	return nil
}

func (r *Repository) GetServiceByKind(kind runnerv1.ServiceKind) (*runnerv1.ExposedService, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	service, exists := r.services[kind]
	if !exists {
		return nil, nil
	}

	return service, nil
}
