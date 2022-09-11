package database

import (
	"bytes"
	"encoding/hex"
)

type Hash [32]byte

func (h Hash) IsEmpty() bool {
	emptyHash := Hash{}

	return bytes.Equal(emptyHash[:], h[:])
}

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(h.Hex()), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

func (h Hash) Hex() string {
	return hex.EncodeToString(h[:])
}
