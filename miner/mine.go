package miner

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"the-blockchain-bar/database"
	"the-blockchain-bar/utils"
	"time"
)

type PendingBlock struct {
	parent database.Hash
	number uint64
	time   uint64
	txs    []database.Tx
}

func NewPendingBlock(parent database.Hash, number uint64, txs []database.Tx) PendingBlock {
	return PendingBlock{parent: parent, number: number, time: uint64(time.Now().Unix()), txs: txs}
}

func Mine(ctx context.Context, pb PendingBlock) (database.Block, error) {
	if len(pb.txs) == 0 {
		return database.Block{}, errors.New("mining empty blocks is not allowed")
	}

	start := time.Now()
	attempt := 0
	var block database.Block
	var hash database.Hash
	var nonce uint32

	for !database.IsBlockHashValid(hash) {
		select {
		case <-ctx.Done():
			fmt.Println("mining cancelled")

			return database.Block{}, fmt.Errorf("mining cancelled: %s", ctx.Err())
		default:
		}

		attempt++
		nonce = generateNonce()

		if attempt%1e+6 == 0 || attempt == 1 {
			fmt.Printf("mining %d pending transactions. attempt no: %d\n", len(pb.txs), attempt)
		}

		block = database.NewBlock(pb.parent, pb.number, nonce, pb.time, pb.txs)
		blockHash, err := block.Hash()
		if err != nil {
			return database.Block{}, fmt.Errorf("could not mine block: %s", err.Error())
		}

		hash = blockHash
	}

	fmt.Printf("\nmined new block '%x' using proof-of-work %s:\n", hash, utils.Unicode("\\U1F389"))
	fmt.Printf("\theight: '%v'\n", pb.number)
	fmt.Printf("\tnonce: '%v'\n", nonce)
	fmt.Printf("\tcreated: '%v'\n", pb.time)
	fmt.Printf("\tparent: '%v'\n\n", pb.parent.Hex())
	fmt.Printf("\tattempt: '%v'\n", attempt)
	fmt.Printf("\ttime: %s\n\n", time.Since(start))

	return block, nil
}

func generateNonce() uint32 {
	rand.Seed(time.Now().UTC().UnixNano())

	return rand.Uint32()
}
