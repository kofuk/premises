package scraper

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

type ProcPidStat struct {
	Utime     uint64
	Stime     uint64
	Starttime uint64
}

func ScrapeProcPidStat(pid int) (*ProcPidStat, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return nil, err
	}
	return ParseProcPidStat(data)
}

func ParseProcPidStat(data []byte) (*ProcPidStat, error) {
	rparenIndex := bytes.IndexByte(data, ')')
	if rparenIndex == -1 {
		return nil, fmt.Errorf("invalid stat format: missing ')'")
	}

	fields := bytes.Fields(data[rparenIndex+2:]) // Skip ") "
	if len(fields) < 20 {
		return nil, fmt.Errorf("invalid stat format: not enough fields")
	}

	utime, err := strconv.ParseUint(string(fields[11]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid utime: %w", err)
	}

	stime, err := strconv.ParseUint(string(fields[12]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid stime: %w", err)
	}

	starttime, err := strconv.ParseUint(string(fields[19]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid starttime: %w", err)
	}

	return &ProcPidStat{
		Utime:     utime,
		Stime:     stime,
		Starttime: starttime,
	}, nil
}
