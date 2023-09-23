package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedPassword(t *testing.T) {
	testcases := []struct {
		name     string
		password string
		allowed  bool
	}{
		{
			name:     "8 chars",
			password: "abcd1234",
			allowed:  true,
		},
		{
			name:     "7 chars",
			password: "abcd123",
			allowed:  false,
		},
		{
			name:     "alphabet only",
			password: "abcdefgh",
			allowed:  false,
		},
		{
			name:     "numeric only",
			password: "12345678",
			allowed:  false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := isAllowedPassword(tt.password)
			assert.Equal(t, tt.allowed, result, fmt.Sprintf("Resut for %s is correct", tt.password))
		})
	}
}
