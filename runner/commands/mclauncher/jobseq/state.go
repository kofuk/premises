package jobseq

import (
	"context"
	"fmt"
	"sync"
)

type State struct {
	waitForConfirmation func(ctx context.Context, state *State) bool
	waitForUpdate       func(state *State)

	confirmations int
	revision      int
	notify        sync.Cond

	L sync.Mutex
}

type StateOption func(state *State)

func NewState(options ...StateOption) *State {
	state := &State{}

	for _, opt := range options {
		opt(state)
	}

	state.L.Lock()

	return state
}

func WaitForConfirmationFunction(fn func(ctx context.Context, state *State) bool) StateOption {
	return func(state *State) {
		state.waitForConfirmation = fn
	}
}

func WaitForUpdateFunction(fn func(state *State)) StateOption {
	return func(state *State) {
		state.waitForUpdate = fn
	}
}

// Wait until confirmed or context is cancelled, and returns whether confirmed or not.
// If false is returned, it indicate the state is updated instead of confirmed.
// If context is cancelled before user action, it's regarded to be confirmed.
func (self *State) WaitForConfirmation(ctx context.Context) bool {
	if self.waitForConfirmation != nil {
		return self.waitForConfirmation(ctx, self)
	}

	self.L.Unlock()
	defer self.L.Lock()

	revision := self.revision
	confirmations := self.confirmations

	for {
		self.notify.Wait()
		if self.revision != revision || self.confirmations != confirmations {
			break
		}
	}

	return self.confirmations == confirmations
}

func (self *State) WaitForUpdate() {
	if self.waitForUpdate != nil {
		self.waitForUpdate(self)
		return
	}

	self.L.Unlock()
	defer self.L.Lock()

	revision := self.revision

	for {
		self.notify.Wait()
		if self.revision != revision {
			break
		}
	}
}

func (self *State) Confirm() error {
	if !self.L.TryLock() {
		return fmt.Errorf("Unable to aquire lock")
	}
	defer self.L.Unlock()

	self.confirmations++
	self.notify.Signal()

	return nil
}

// This function requires the state to be locked
func (self *State) NotifyUpdate() {
	self.revision++
	self.notify.Signal()
}
