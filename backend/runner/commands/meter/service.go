package meter

import (
	"context"
	"errors"
	"sync"

	"github.com/kofuk/premises/backend/common/util"
	"github.com/kofuk/premises/backend/runner/commands/meter/scraper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sys/unix"
)

const ClkTck = 100

type MeterService struct {
	targets map[int]struct{}
	m       sync.Mutex
}

func NewMeterService() *MeterService {
	return &MeterService{
		targets: make(map[int]struct{}),
	}
}

func getCurrentMonotonicTimeSec() (float64, error) {
	var t unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &t); err != nil {
		return 0, err
	}
	return float64(t.Sec) + float64(t.Nsec)/1e9, nil
}

func (s *MeterService) Initialize() error {
	meter := otel.Meter("meter")

	cpuQuota, cpuPeriod, err := scraper.GetCPUQuota()
	if err != nil {
		return err
	}

	util.Must(meter.Float64ObservableCounter("premises.runner.host.cpu",
		metric.WithDescription("Total CPU time available on the host in seconds since process start"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
			targets := s.getAllTargets()

			var (
				currentTimeSec float64
				errs           []error
			)

			for i, pid := range targets {
				if i%10 == 0 {
					if time, err := getCurrentMonotonicTimeSec(); err != nil {
						return err
					} else {
						currentTimeSec = time
					}
				}

				data, err := scraper.ScrapeProcPidStat(pid)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				startTimeSec := float64(data.Starttime) / ClkTck
				if startTimeSec > 0 {
					o.Observe(
						(currentTimeSec-startTimeSec)*float64(cpuQuota)/float64(cpuPeriod),
						metric.WithAttributes(attribute.Int("process.pid", pid)),
					)
				}
			}

			return errors.Join(errs...)
		}),
	))

	util.Must(meter.Float64ObservableCounter("premises.runner.minecraft.cpu",
		metric.WithDescription("Total CPU time used by Minecraft processes in seconds since process start"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
			targets := s.getAllTargets()

			var errs []error
			for _, pid := range targets {
				data, err := scraper.ScrapeProcPidStat(pid)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				startTimeSec := float64(data.Starttime) / ClkTck
				if startTimeSec > 0 {
					cpuTimeSec := (float64(data.Utime) + float64(data.Stime)) / ClkTck
					o.Observe(
						cpuTimeSec,
						metric.WithAttributes(attribute.Int("process.pid", pid)),
					)
				}
			}

			return errors.Join(errs...)
		}),
	))

	util.Must(meter.Int64ObservableGauge("premises.runner.cpu.frequency",
		metric.WithDescription("Current CPU frequency in hertz"),
		metric.WithUnit("Hz"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			cpus, err := scraper.ScrapeOnlineCPUs()
			if err != nil {
				return err
			}

			var errs []error
			for _, cpu := range cpus {
				freq, err := scraper.ScrapeCPUScalingCurFreq(cpu)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				o.Observe(
					int64(freq)*1000, // Convert kHz to Hz
					metric.WithAttributes(attribute.Int("cpu.logical_number", cpu)),
				)
			}

			return errors.Join(errs...)
		}),
	))

	util.Must(meter.Int64ObservableGauge("premises.runner.cpu.frequency.max",
		metric.WithDescription("Maximum CPU frequency in hertz"),
		metric.WithUnit("Hz"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			cpus, err := scraper.ScrapeOnlineCPUs()
			if err != nil {
				return err
			}

			var errs []error
			for _, cpu := range cpus {
				freq, err := scraper.ScrapeCPUScalingMaxFreq(cpu)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				o.Observe(
					int64(freq)*1000, // Convert kHz to Hz
					metric.WithAttributes(attribute.Int("cpu.logical_number", cpu)),
				)
			}

			return errors.Join(errs...)
		}),
	))

	return nil
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

func (s *MeterService) getAllTargets() []int {
	s.m.Lock()
	defer s.m.Unlock()
	targets := make([]int, 0, len(s.targets))
	for pid := range s.targets {
		targets = append(targets, pid)
	}
	return targets
}
