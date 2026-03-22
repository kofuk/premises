package scraper

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

func ScrapeOnlineCPUs() ([]int, error) {
	data, err := os.ReadFile("/sys/devices/system/cpu/online")
	if err != nil {
		return nil, err
	}

	return ParseOnlineCPUs(data)
}

func ParseOnlineCPUs(data []byte) ([]int, error) {
	var cpus []int
	parts := bytes.Split(bytes.TrimSpace(data), []byte{','})
	for _, part := range parts {
		if bytes.Contains(part, []byte{'-'}) {
			rangeParts := bytes.Split(part, []byte{'-'})
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid cpu range: %s", part)
			}

			start, err := strconv.Atoi(string(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid cpu number: %w", err)
			}

			end, err := strconv.Atoi(string(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid cpu number: %w", err)
			}

			for i := start; i <= end; i++ {
				cpus = append(cpus, i)
			}
		} else {
			cpu, err := strconv.Atoi(string(part))
			if err != nil {
				return nil, fmt.Errorf("invalid cpu number: %w", err)
			}
			cpus = append(cpus, cpu)
		}
	}

	return cpus, nil
}

func ScrapeCPUfreq(cpu int, name string) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/%s", cpu, name))
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseUint(string(bytes.TrimSpace(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid cpufreq value: %w", err)
	}

	return value, nil
}

func ScrapeCPUScalingMaxFreq(cpu int) (uint64, error) {
	return ScrapeCPUfreq(cpu, "scaling_max_freq")
}

func ScrapeCPUScalingCurFreq(cpu int) (uint64, error) {
	return ScrapeCPUfreq(cpu, "scaling_cur_freq")
}
