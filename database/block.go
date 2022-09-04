package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Hash [32]byte

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

type Block struct {
	Header BlockHeader `json:"header"`  // metadata (parent block hash + timestamp)
	TXs    []Tx        `json:"payload"` // new transactions only (payload)
}

type BlockHeader struct {
	Parent Hash   `json:"parent"` // parent block reference
	Number uint64 `json:"number"`
	Time   uint64 `json:"time"`
}

func NewBlock(parent Hash, number, time uint64, txs []Tx) Block {
	return Block{BlockHeader{parent, number, time}, txs}
}

func (b Block) Hash() (hash Hash, err error) {
	blockJson, err := json.Marshal(b)
	if err != nil {
		return hash, err
	}

	return sha256.Sum256(blockJson), nil
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}
