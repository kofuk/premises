package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parsePlayerList(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		players      PlayerList
		expectsError bool
	}{
		{
			name:    "No player",
			input:   "There are 0 of a max of 20 players online: ",
			players: nil,
		},
		{
			name:    "One player",
			input:   "There are 1 of a max of 20 players online: test1",
			players: []string{"test1"},
		},
		{
			name:    "Many player",
			input:   "There are 2 of a max of 20 players online: test1, test2",
			players: []string{"test1", "test2"},
		},
		{
			name:         "Mistach",
			input:        "There are 2 of a max of 20 players online: test1, test2, test3",
			expectsError: true,
		},
		{
			name:         "Empty",
			input:        "",
			expectsError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parsePlayerList(tt.input)
			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.players, p)
			}
		})
	}
}

func Test_parseSeed(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		seed         string
		expectsError bool
	}{
		{
			name:  "Normal",
			input: "Seed: [-6288715796049084847]",
			seed:  "-6288715796049084847",
		},
		{
			name:         "Empty",
			input:        "",
			expectsError: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := parseSeed(tt.input)
			if tt.expectsError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.seed, s)
			}
		})
	}
}
