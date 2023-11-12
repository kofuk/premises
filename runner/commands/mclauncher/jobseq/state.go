package jobseq

import "context"

type State struct {
	waitForConfirmation func(ctx context.Context, state *State) bool
	waitForUpdate       func(state *State)
}

type StateOption func(state *State)

func NewState(options ...StateOption) *State {
	state := &State{}

	for _, opt := range options {
		opt(state)
	}

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

	return true
}

func (self *State) WaitForUpdate() {
	if self.waitForUpdate != nil {
		self.waitForUpdate(self)
		return
	}
}
