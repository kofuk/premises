package util

import (
	"io"
	"time"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/exterior"
)

type ProgressReader struct {
	r          io.Reader
	event      entity.EventCode
	total      int
	current    int
	prevUpdate time.Time
}

func NewProgressReader(base io.Reader, event entity.EventCode, total int) *ProgressReader {
	return &ProgressReader{
		r:     base,
		event: event,
		total: total,
	}
}

func (r *ProgressReader) Read(buf []byte) (int, error) {
	n, err := r.r.Read(buf)
	r.current += n
	if err == nil && time.Since(r.prevUpdate) >= time.Second {
		r.prevUpdate = time.Now()

		go func(current int) {
			percent := 100
			if r.total != 0 {
				percent = current * 100 / r.total
			}

			exterior.SendEvent(runner.Event{
				Type: runner.EventStatus,
				Status: &runner.StatusExtra{
					EventCode: r.event,
					Progress:  percent,
				},
			})
		}(r.current)
	}

	return n, err
}

func (r *ProgressReader) ToSeekable() *SeekableProgressReader {
	return &SeekableProgressReader{
		ProgressReader: *r,
	}
}

type SeekableProgressReader struct {
	ProgressReader
}

func (r *SeekableProgressReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart && offset == 0 {
		r.current = 0
	}
	return r.r.(io.Seeker).Seek(offset, whence)
}
