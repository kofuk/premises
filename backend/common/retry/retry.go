package retry

import (
	"log/slog"
	"math/rand"
	"time"
)

type r struct {
	rand        *rand.Rand
	curInterval time.Duration
	maxInterval time.Duration
	failAfter   time.Duration
	elapsedTime time.Duration
}

func (r *r) nextInterval() time.Duration {
	random := 0
	if r.rand == nil {
		random = rand.Intn(3000)
	} else {
		random = r.rand.Intn(3000)
	}

	curInterval := r.curInterval + time.Duration(random)*time.Millisecond

	r.elapsedTime += curInterval

	r.curInterval *= 2
	if r.curInterval > r.maxInterval {
		r.curInterval = r.maxInterval
	}

	return curInterval
}

func (r *r) finished() bool {
	return r.failAfter < r.elapsedTime
}

func (r *r) wait() {
	time.Sleep(r.nextInterval())
}

type Void struct{}

var V = struct{}{}

func Retry[T any](fn func() (T, error), failAfter time.Duration) (T, error) {
	rr := r{
		curInterval: 1 * time.Second,
		maxInterval: 1 * time.Minute,
		failAfter:   failAfter,
	}

	for {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		if rr.finished() {
			return *new(T), err
		}

		slog.Debug("Retrying...", slog.String("error", err.Error()))

		rr.wait()
	}
}
