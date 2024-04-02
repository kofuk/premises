package systemutil

import (
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
	// We don't need to call Close for the file because close(2) for associated fd should be called later.

	buf := make([]byte, 1024)
	remain := int(size)
	for remain >= len(buf) {
		n, err := file.Write(buf)
		if err != nil {
			return err
		}
		remain -= n
	}
	if remain > 0 {
		if _, err := file.Write(buf[:remain]); err != nil {
			return err
		}
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
