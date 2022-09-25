package miner

import (
	"context"
	"testing"
	"the-blockchain-bar/database"
	"time"
)

func TestMine(t *testing.T) {
	pendingBlock := createRandomPendingBlock()

	ctx := context.Background()

	minedBlock, err := Mine(ctx, pendingBlock)
	if err != nil {
		t.Fatal(err)
	}

	minedBlockHash, err := minedBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !database.IsBlockHashValid(minedBlockHash) {
		t.Fatal()
	}
}

func TestMineWithTimeout(t *testing.T) {
	pendingBlock := createRandomPendingBlock()

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err := Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock() PendingBlock {
	return NewPendingBlock(
		database.Hash{},
		0,
		[]database.Tx{
			database.NewTx("andrej", "andrej", 3, ""),
			database.NewTx("andrej", "andrej", 700, "reward"),
		},
	)
}
