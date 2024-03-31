package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type State struct {
	m               sync.Mutex
	PrevWoldVersion string              `json:"-"`
	WorldVersion    string              `json:"worldVersion"`
	WhitelistUsers  map[string]struct{} `json:"whitelist"`
	OpUsers         map[string]struct{} `json:"op"`
}

func newState() *State {
	return &State{
		WorldVersion: uuid.NewString(),
	}
}

func CreateState() (*State, error) {
	f, err := os.Open("world/.fake_state")
	if err != nil {
		if os.IsNotExist(err) {
			return newState(), nil
		}
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	state := new(State)
	if err := dec.Decode(state); err != nil {
		return newState(), nil
	}
	state.PrevWoldVersion = state.WorldVersion
	state.WorldVersion = uuid.NewString()

	return state, nil
}

func (s *State) Save() error {
	if err := os.MkdirAll("world", 0755); err != nil {
		return err
	}

	f, err := os.Create("world/.fake_state")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(s); err != nil {
		return err
	}
	return nil
}

func (s *State) AddToWhitelist(user string) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.WhitelistUsers == nil {
		s.WhitelistUsers = make(map[string]struct{})
	}

	s.WhitelistUsers[user] = struct{}{}
}

func (s *State) AddToOp(user string) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.OpUsers == nil {
		s.OpUsers = make(map[string]struct{})
	}

	s.OpUsers[user] = struct{}{}
}

type PublicState struct {
	ServerName       string   `json:"version"`
	PrevWorldVersion string   `json:"worldVersionPrev"`
	WorldVersion     string   `json:"worldVersion"`
	Whitelist        []string `json:"whitelist"`
	Op               []string `json:"op"`
}

func (s *State) ToPublicState() PublicState {
	s.m.Lock()
	defer s.m.Unlock()

	whitelist := make([]string, 0)
	op := make([]string, 0)

	for k := range s.WhitelistUsers {
		whitelist = append(whitelist, k)
	}
	for k := range s.OpUsers {
		op = append(op, k)
	}

	serverName := strings.TrimSuffix(filepath.Base(os.Args[0]), ".jar")

	return PublicState{
		ServerName:       serverName,
		PrevWorldVersion: s.PrevWoldVersion,
		WorldVersion:     s.WorldVersion,
		Whitelist:        whitelist,
		Op:               op,
	}
}
