package miner

import (
	"context"
	"testing"
	"the-blockchain-bar/database"
	"the-blockchain-bar/wallet"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestMine(t *testing.T) {
	miner := database.NewAccount(wallet.AndrejAccount)
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

	if minedBlock.Header.Miner.String() != miner.String() {
		t.Fatal("mined block miner should equal miner from pending block")
	}
}

func TestMineWithTimeout(t *testing.T) {
	miner := database.NewAccount(wallet.AndrejAccount)
	pendingBlock := createRandomPendingBlock(miner)

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err := Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock(miner common.Address) PendingBlock {
	return NewPendingBlock(
		database.Hash{},
		1,
		miner,
		[]database.Tx{
			{
				From:  database.NewAccount(wallet.AndrejAccount),
				To:    database.NewAccount(wallet.BabaYagaAccount),
				Value: 1,
				Time:  1579451695,
				Data:  "",
			},
		},
	)
}
