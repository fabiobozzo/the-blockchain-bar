package database

import (
	"encoding/hex"
	"testing"

	"github.com/test-go/testify/assert"
)

func TestIsBlockHashValid(t *testing.T) {
	testCases := map[string]struct {
		hexHash   string
		wantValid bool
	}{
		"valid": {
			hexHash:   "000000fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa",
			wantValid: true,
		},
		"invalid": {
			hexHash:   "010001fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa",
			wantValid: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var hash = Hash{}
			_, err := hex.Decode(hash[:], []byte(tc.hexHash))
			assert.NoError(t, err)

			assert.Equal(t, tc.wantValid, IsBlockHashValid(hash))
		})
	}
}
