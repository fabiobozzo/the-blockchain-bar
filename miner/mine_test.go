package miner

import (
	"context"
	"testing"
	"the-blockchain-bar/database"
	"time"
)

func TestMine(t *testing.T) {
	miner := database.NewAccount("andrej")
	pendingBlock := createRandomPendingBlock(miner)

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

	if minedBlock.Header.Miner != miner {
		t.Fatal("mined block miner should equal miner from pending block")
	}
}

func TestMineWithTimeout(t *testing.T) {
	miner := database.NewAccount("andrej")
	pendingBlock := createRandomPendingBlock(miner)

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err := Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock(miner database.Account) PendingBlock {
	return NewPendingBlock(
		database.Hash{},
		1,
		miner,
		[]database.Tx{
			database.Tx{From: "andrej", To: "babayaga", Value: 1, Time: 1579451695, Data: ""},
		},
	)
}
