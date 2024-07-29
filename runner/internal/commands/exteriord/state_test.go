package exteriord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type inMemoryBackend struct {
	s map[string]string
}

func (b *inMemoryBackend) LoadStates() (map[string]string, error) {
	return b.s, nil
}

func (b *inMemoryBackend) SaveStates(s map[string]string) error {
	b.s = s
	return nil
}

func Test_StateStore_Set(t *testing.T) {
	backend := &inMemoryBackend{
		s: make(map[string]string),
	}
	sut := NewStateStore(backend)

	err := sut.Set("foo", "111")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"foo": "111"}, backend.s)

	err = sut.Set("foo", "222")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"foo": "222"}, backend.s)
}

func Test_StateStore_Get(t *testing.T) {
	backend := &inMemoryBackend{
		s: make(map[string]string),
	}
	sut := NewStateStore(backend)

	value, err := sut.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, "", value)

	sut.Set("foo", "111")
	value, err = sut.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, "111", value)
}

func Test_StateStore_Remove(t *testing.T) {
	backend := &inMemoryBackend{
		s: make(map[string]string),
	}
	sut := NewStateStore(backend)

	err := sut.Remove("foo")
	assert.NoError(t, err)

	sut.Set("foo", "111")
	sut.Set("bar", "222")

	err = sut.Remove("foo")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"bar": "222"}, backend.s)
}
