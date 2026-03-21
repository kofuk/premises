package meter

import (
	"sync"

	"go.opentelemetry.io/otel"
)

type MeterService struct {
	targets map[int]struct{}
	m       sync.Mutex
}

func NewMeterService() *MeterService {
	return &MeterService{
		targets: make(map[int]struct{}),
	}
}

func (s *MeterService) Initialize() {
	meter := otel.Meter("meter")

	_ = meter
}

func (s *MeterService) RegisterTarget(pid int) {
	s.m.Lock()
	s.targets[pid] = struct{}{}
	s.m.Unlock()
}

func (s *MeterService) UnregisterTarget(pid int) {
	s.m.Lock()
	delete(s.targets, pid)
	s.m.Unlock()
}
