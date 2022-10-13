package database

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const BlockReward = 100

type Block struct {
	Header BlockHeader `json:"header"`  // metadata (parent block hash + timestamp)
	TXs    []SignedTx  `json:"payload"` // new transactions only (payload)
}

type BlockHeader struct {
	Parent Hash           `json:"parent"` // parent block reference
	Number uint64         `json:"number"`
	Nonce  uint32         `json:"nonce"`
	Time   uint64         `json:"time"`
	Miner  common.Address `json:"miner"`
}

func NewBlock(parent Hash, number uint64, nonce uint32, time uint64, miner common.Address, txs []SignedTx) Block {
	return Block{BlockHeader{parent, number, nonce, time, miner}, txs}
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

func IsBlockHashValid(hash Hash) bool {
	return fmt.Sprintf("%x", hash[0]) == "0" &&
		fmt.Sprintf("%x", hash[1]) == "0" &&
		fmt.Sprintf("%x", hash[2]) == "0" &&
		fmt.Sprintf("%x", hash[3]) != "0"
}
