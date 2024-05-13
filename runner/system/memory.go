package system

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

func parseMeminfoLine(line string) (string, int, error) {
	pos := strings.Index(line, ":")
	if pos < 0 {
		return "", 0, errors.New("invalid line")
	}

	key := line[0:pos]
	value := strings.Trim(line[pos+1:], " \r\n")
	fields := strings.Split(value, " ")
	var intVal int
	if len(fields) == 0 {
		return key, 0, errors.New("parse error")
	}

	var err error
	intVal, err = strconv.Atoi(fields[0])
	if err != nil {
		return key, 0, err
	}

	if len(fields) > 1 {
		if fields[1] == "kB" {
			intVal *= 1024
		} else {
			return key, 0, errors.New("unsupported suffix")
		}
	}

	return key, intVal, nil
}

func readTotalMemoryFromReader(meminfo io.Reader) (int, error) {
	r := bufio.NewReader(meminfo)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				return 0, errors.New("reached to EOF without reading MemTotal")
			}
		}

		key, value, err := parseMeminfoLine(string(line))
		if err != nil {
			if key != "MemTotal" {
				// If it is not MemTotal line, we ignore the error because MemTotal line may be valid.
				continue
			}
			return 0, err
		}

		if key == "MemTotal" {
			return value, nil
		}
	}
}

func GetTotalMemory() (int, error) {
	meminfo, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer meminfo.Close()

	return readTotalMemoryFromReader(meminfo)
}
