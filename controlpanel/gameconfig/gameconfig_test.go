package gameconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_addToSlice(t *testing.T) {
	testcases := []struct {
		name    string
		source  []string
		element string
		result  []string
	}{
		{
			name:    "Should appended",
			source:  []string{"hoge", "fuga"},
			element: "piyo",
			result:  []string{"hoge", "fuga", "piyo"},
		},
		{
			name:    "Should not appended",
			source:  []string{"hoge", "fuga"},
			element: "fuga",
			result:  []string{"hoge", "fuga"},
		},
	}
	for _, tt := range testcases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			result := addToSlice(tt.source, tt.element)
			assert.Equal(t, tt.result, result)
		})
	}
}

func Test_calculateMemSizeForGame(t *testing.T) {
	testcases := []struct {
		name          string
		size          int
		shouldBeError bool
		expectedSize  int
	}{
		{
			name:          "too small",
			size:          2047,
			shouldBeError: true,
		},
		{
			name:         "standard",
			size:         2048,
			expectedSize: 1024,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			size, err := calculateMemSizeForGame(tt.size)
			if tt.shouldBeError {
				assert.ErrorIs(t, err, ErrMemoryTooSmall)
			} else {
				assert.Equal(t, tt.expectedSize, size)
			}
		})
	}
}

func Test_isValidLevelType(t *testing.T) {
	testcases := []struct {
		levelType string
		isValid   bool
	}{
		{
			levelType: "default",
			isValid:   true,
		},
		{
			levelType: "flat",
			isValid:   true,
		},
		{
			levelType: "largeBiomes",
			isValid:   true,
		},
		{
			levelType: "amplified",
			isValid:   true,
		},
		{
			levelType: "buffet",
			isValid:   true,
		},
		{
			levelType: "",
			isValid:   false,
		},
		{
			levelType: "hoge",
			isValid:   false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.levelType, func(t *testing.T) {
			isValid := isValidLevelType(tt.levelType)
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func Test_isValidDifficulty(t *testing.T) {
	testcases := []struct {
		difficulty string
		isValid    bool
	}{
		{
			difficulty: "peaceful",
			isValid:    true,
		},
		{
			difficulty: "easy",
			isValid:    true,
		},
		{
			difficulty: "normal",
			isValid:    true,
		},
		{
			difficulty: "hard",
			isValid:    true,
		},
		{
			difficulty: "",
			isValid:    false,
		},
		{
			difficulty: "hoge",
			isValid:    false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.difficulty, func(t *testing.T) {
			isValid := isValidDifficulty(tt.difficulty)
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}
