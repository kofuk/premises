package scraper

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

type ProcPidIO struct {
	ReadBytes           uint64
	WriteBytes          uint64
	CancelledWriteBytes uint64
}

func ScrapeProcPidIO(pid int) (*ProcPidIO, error) {
	file, err := os.Open(fmt.Sprintf("/proc/%d/io", pid))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseProcPidIO(file)
}

func ParseProcPidIO(source io.Reader) (*ProcPidIO, error) {
	reader := bufio.NewReader(source)
	result := &ProcPidIO{}
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		parts := bytes.SplitN(line, []byte{':'}, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid io format: %s", line)
		}

		key := bytes.TrimSpace(parts[0])
		valueStr := bytes.TrimSpace(parts[1])
		value, err := strconv.ParseUint(string(valueStr), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid io value: %w", err)
		}

		switch string(key) {
		case "read_bytes":
			result.ReadBytes = value
		case "write_bytes":
			result.WriteBytes = value
		case "cancelled_write_bytes":
			result.CancelledWriteBytes = value
		}
	}

	return result, nil
}
