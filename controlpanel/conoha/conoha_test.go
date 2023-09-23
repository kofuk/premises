package conoha

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getSpecFromFlavorName(t *testing.T) {
	testcases := []struct {
		name       string
		flavorName string
		cores      int
		mem        int
		disk       int
		err        error
	}{
		{
			name:       "normal1",
			flavorName: "g-c2m1d100",
			cores:      2,
			mem:        1,
			disk:       100,
		},
		{
			name:       "normal2",
			flavorName: "g-c24m64d100",
			cores:      24,
			mem:        64,
			disk:       100,
		},
		{
			name:       "unsupported1",
			flavorName: "32gb-flavor",
			err:        unsupportedFlavorError,
		},
		{
			name:       "unsupported2",
			flavorName: "g-cmd",
			err:        unsupportedFlavorError,
		},
		{
			name:       "unsupported3",
			flavorName: "g-c1md",
			err:        unsupportedFlavorError,
		},
		{
			name:       "unsupported4",
			flavorName: "g-c1m1d",
			err:        unsupportedFlavorError,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			cores, mem, disk, err := getSpecFromFlavorName(testcase.flavorName)
			if testcase.err != nil {
				assert.ErrorIs(t, err, testcase.err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, testcase.cores, cores, "Core matches")
			assert.Equal(t, testcase.mem, mem, "Mem matches")
			assert.Equal(t, testcase.disk, disk, "Disk matches")
		})
	}
}
