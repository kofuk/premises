package backup

import "io"

type ProgressReader struct {
	reader io.Reader
	notify chan int
}

func (self *ProgressReader) Read(buf []byte) (int, error) {
	n, err := self.reader.Read(buf)
	if err == nil {
		self.notify <- n
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
	}
	return n, err
}
