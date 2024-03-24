package util

import (
	"io"
	"log/slog"
)

type ProgressReader struct {
	R  io.Reader
	Ch chan int
}

func (self *ProgressReader) Read(buf []byte) (int, error) {
	n, err := self.R.Read(buf)
	if err == nil {
		self.Ch <- n
		slog.Debug("read", slog.Int("byte_count", n))
	}
	return n, err
}

type ProgressWriter struct {
	W  io.Writer
	Ch chan int
}

func (self *ProgressWriter) Write(buf []byte) (int, error) {
	n, err := self.W.Write(buf)
	if err == nil {
		self.Ch <- n
		slog.Debug("write", slog.Int("byte_count", n))
	}
	return n, err
}
