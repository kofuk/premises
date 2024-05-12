package systemutil

import (
	"errors"

	"github.com/mackerelio/go-osstat/cpu"
)

type CPUUsage struct {
	prevTotal uint64
	prevIdle  uint64
}

func NewCPUUsage() (*CPUUsage, error) {
	c := &CPUUsage{}
	if _, err := c.Percent(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *CPUUsage) Percent() (float64, error) {
	usage, err := cpu.Get()
	if err != nil {
		return 0, err
	}

	total := usage.Total
	idle := usage.Idle

	diffTotal := total - c.prevTotal
	if diffTotal <= 0 {
		return 0, errors.New("try again later")
	}
	percentIdle := float64(idle-c.prevIdle) / float64(total-c.prevTotal) * 100

	c.prevTotal = total
	c.prevIdle = idle

	return 100.0 - percentIdle, nil
}
