package exteriord

import (
	"encoding/json"
	"os"
	"sync"
)

type StateBackend interface {
	LoadStates() (map[string]string, error)
	SaveStates(states map[string]string) error
}

type StateStore struct {
	m       sync.Mutex
	path    string
	backend StateBackend
}

type LocalStorageStateBackend struct {
	path string
}

func NewLocalStorageStateBackend(path string) *LocalStorageStateBackend {
	return &LocalStorageStateBackend{
		path: path,
	}
}

func NewStateStore(backend StateBackend) *StateStore {
	return &StateStore{
		backend: backend,
	}
}

func (b *LocalStorageStateBackend) LoadStates() (map[string]string, error) {
	file, err := os.Open(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	defer file.Close()

	var state map[string]string
	dec := json.NewDecoder(file)
	if err := dec.Decode(&state); err != nil {
		return nil, err
	}

	return state, nil
}

func (b *LocalStorageStateBackend) SaveStates(state map[string]string) error {
	file, err := os.Create(b.path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if err := enc.Encode(state); err != nil {
		return err
	}

	return nil
}

func (s *StateStore) Set(key, value string) error {
	s.m.Lock()
	defer s.m.Unlock()

	state, err := s.backend.LoadStates()
	if err != nil {
		return err
	}

	state[key] = value

	if err := s.backend.SaveStates(state); err != nil {
		return err
	}

	return nil
}

func (s *StateStore) Get(key string) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	state, err := s.backend.LoadStates()
	if err != nil {
		return "", err
	}

	return state[key], nil
}

func (s *StateStore) Remove(key string) error {
	s.m.Lock()
	defer s.m.Unlock()

	state, err := s.backend.LoadStates()
	if err != nil {
		return err
	}

	delete(state, key)

	if err := s.backend.SaveStates(state); err != nil {
		return err
	}

	return nil
}
