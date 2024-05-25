package retry

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createReferenceInterval() []time.Duration {
	intervals := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		32 * time.Second,
		1 * time.Minute,
		1 * time.Minute,
	}
	rand := rand.New(rand.NewSource(0))
	for i := 0; i < len(intervals); i++ {
		intervals[i] += time.Duration(rand.Intn(3000)) * time.Millisecond
	}

	return intervals
}

func Test_calculateInterval(t *testing.T) {
	intervals := createReferenceInterval()
	sut := r{
		rand:        rand.New(rand.NewSource(0)),
		curInterval: 1 * time.Second,
		maxInterval: 1 * time.Minute,
		failAfter:   3 * time.Minute,
	}

	for _, expectedInterval := range intervals {
		assert.False(t, sut.finished())
		got := sut.nextInterval()
		assert.Equal(t, expectedInterval, got)
	}
	assert.True(t, sut.finished())
}
