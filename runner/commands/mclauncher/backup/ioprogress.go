package backup

import (
	"io"
	"log/slog"
)

type ProgressReader struct {
	reader io.Reader
	notify chan int
}

func (self *ProgressReader) Read(buf []byte) (int, error) {
	n, err := self.reader.Read(buf)
	if err == nil {
		self.notify <- n
		slog.Debug("read", slog.Int("byte_count", n))
	}
	return n, err
}

type ProgressWriter struct {
	writer io.Writer
	notify chan int
}

func (self *ProgressWriter) Write(buf []byte) (int, error) {
	n, err := self.writer.Write(buf)
	if err == nil {
		self.notify <- n
		slog.Debug("write", slog.Int("byte_count", n))
	}
	return n, err
}
