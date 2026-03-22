package scraper

import (
	"bytes"
	"errors"
	"os"
	"runtime"
	"strconv"
)

func GetCPUQuota() (uint64, uint64, error) {
	// cgroup v2
	cpuMaxData, err := os.ReadFile("/sys/fs/cgroup/cpu.max")
	if err != nil && os.IsNotExist(err) {
		// Not in cgroup v2 environment, fallback to runtime.NumCPU()
		return uint64(runtime.NumCPU()), 1, nil
	}

	fields := bytes.Fields(cpuMaxData)
	if len(fields) != 2 {
		// Unexpected format, fallback to runtime.NumCPU()
		return 0, 0, errors.New("malformed cpu.max data")
	}

	if bytes.Equal(fields[0], []byte("max")) {
		// No CPU limit, return the number of CPUs on the host
		return uint64(runtime.NumCPU()), 1, nil
	}

	// Parse the CPU quota
	cpuQuota, err := strconv.ParseUint(string(fields[0]), 10, 64)
	if err != nil {
		// Failed to parse CPU quota, fallback to runtime.NumCPU()
		return 0, 0, err
	}

	// Parse the CPU period
	cpuPeriod, err := strconv.ParseUint(string(fields[1]), 10, 64)
	if err != nil {
		// Failed to parse CPU period, fallback to runtime.NumCPU()
		return 0, 0, err
	}

	if cpuPeriod == 0 {
		// Avoid division by zero, fallback to runtime.NumCPU()
		return uint64(runtime.NumCPU()), 1, nil
	}

	return cpuQuota, cpuPeriod, nil
}
