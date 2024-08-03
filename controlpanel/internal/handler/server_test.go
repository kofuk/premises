package handler

import (
	"testing"

	"github.com/kofuk/premises/controlpanel/internal/conoha"
	"github.com/stretchr/testify/assert"
)

func Test_findMatchingFlavor(t *testing.T) {
	flavors := []conoha.Flavor{
		{ID: "1", RAM: 8192},
		{ID: "2", RAM: 4096},
		{ID: "3", RAM: 1024},
		{ID: "4", RAM: 2048},
	}

	t.Run("found", func(t *testing.T) {
		id, err := findMatchingFlavor(flavors, 2048)
		assert.NoError(t, err)
		assert.Equal(t, "4", id)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findMatchingFlavor(flavors, 3000)
		assert.Error(t, err, "no matching flavor")
	})
}
