package systemutil

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

func Fallocate(path string, size int64) error {
	fd, err := syscall.Creat(path, 0644)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	if err := syscall.Fallocate(fd, 0, 0, size); err == nil {
		return nil
	} else if err != syscall.EOPNOTSUPP {
		return err
	}

	slog.Debug("The filesystem seems not to support fallocate(2); Will fallback to write(2) to create a file with specified length")

	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		return errors.New("Bad file descriptor")
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	buf := make([]byte, 1024)
	written := 0
	for int(size)-written >= len(buf) {
		if _, err := writer.Write(buf); err != nil {
			return err
		}
	}
	if _, err := writer.Write(buf[:int(size)-written]); err != nil {
		return err
	}

	return nil
}

func ChownRecursive(path string, uid, gid int) error {
	var errs []error

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		slog.Debug(fmt.Sprintf("Changing ownership of file: %s", path))

		if err := syscall.Chown(path, uid, gid); err != nil {
			errs = append(errs, err)
		}

		return err
	})
	if err != nil {
		return err
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
