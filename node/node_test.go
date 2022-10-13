package node

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"the-blockchain-bar/miner"
	"the-blockchain-bar/wallet"
	"time"

	"github.com/test-go/testify/require"

	"the-blockchain-bar/database"
	"the-blockchain-bar/utils"
)

func TestNode_Run(t *testing.T) {
	dataDir := getTestDataDirPath()
	if err := utils.RemoveDir(dataDir); err != nil {
		t.Fatal(err)
	}

	http.DefaultServeMux = new(http.ServeMux)

	n := New(dataDir, "127.0.0.1", 8085, database.NewAccount(wallet.AndrejAccount), PeerNode{})

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)

	if err := n.Run(ctx); err != http.ErrServerClosed {
		t.Fatalf("node server was suppose to close after 5s, instead: %s", err)
	}
}

func TestNode_Mining(t *testing.T) {
	andrej := database.NewAccount(wallet.AndrejAccount)
	babayaga := database.NewAccount(wallet.BabaYagaAccount)

	dataDir := getTestDataDirPath()
	if err := utils.RemoveDir(dataDir); err != nil {
		t.Fatal(err)
	}

	http.DefaultServeMux = new(http.ServeMux)

	n := New(dataDir, "127.0.0.1", 8085, andrej, PeerNode{})
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)
	myselfNode := NewPeerNode("127.0.0.1", 8085, false, database.NewAccount(""), true)

	go func() {
		time.Sleep(time.Second * 1)
		tx := database.NewTx(andrej, babayaga, 1, "")

		require.NoError(t, n.AddPendingTX(tx, myselfNode))
	}()

	go func() {
		time.Sleep(time.Second * 30)
		tx := database.NewTx(andrej, babayaga, 2, "")

		require.NoError(t, n.AddPendingTX(tx, myselfNode))
	}()

	go func() {
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ticker.C:
				if n.state.LatestBlock().Header.Number == 2 {
					closeNode()
					return
				}
			}
		}
	}()

	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Number != 2 {
		t.Fatal("was suppose to mine 2 pending tx into 2 valid blocks under 30m")
	}
}

func TestNode_MiningStopsOnNewSyncedBlock(t *testing.T) {
	andrej := database.NewAccount(wallet.AndrejAccount)
	babayaga := database.NewAccount(wallet.BabaYagaAccount)

	dataDir := getTestDataDirPath()
	if err := utils.RemoveDir(dataDir); err != nil {
		t.Fatal(err)
	}

	// Required for AddPendingTX() to describe from what node the TX came from (local node in this case)
	localNode := NewPeerNode(
		"127.0.0.1",
		8085,
		false,
		database.NewAccount(""),
		true,
	)

	http.DefaultServeMux = new(http.ServeMux)
	n := New(dataDir, "127.0.0.1", 8085, babayaga, PeerNode{})
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)

	tx1 := database.Tx{From: andrej, To: babayaga, Value: 1, Time: 1579451695, Data: ""}
	tx2 := database.NewTx(andrej, babayaga, 2, "")
	tx2Hash, _ := tx2.Hash()

	validPreMinedPb := miner.NewPendingBlock(database.Hash{}, 0, andrej, []database.Tx{tx1})
	validSyncedBlock, err := miner.Mine(ctx, validPreMinedPb)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds - 2))

		if err := n.AddPendingTX(tx1, localNode); err != nil {
			t.Fatal(err)
		}

		if err := n.AddPendingTX(tx2, localNode); err != nil {
			t.Fatal(err)
		}
	}()

	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("the node should be mining")
		}

		if _, err := n.state.AddBlock(validSyncedBlock); err != nil {
			t.Fatal(err)
		}

		n.newSyncedBlocks <- validSyncedBlock

		time.Sleep(time.Second * 2)
		if n.isMining {
			t.Fatal("new received block should have canceled mining")
		}

		_, onlyTX2IsPending := n.pendingTXs[tx2Hash.Hex()]

		if len(n.pendingTXs) != 1 && !onlyTX2IsPending {
			t.Fatal("new received block should have canceled mining of already mined transaction")
		}

		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("should be mining again the 1 TX not included in synced block")
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
				if n.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)

		startingAndrejBalance := n.state.Balances[andrej]
		startingBabayagaBalance := n.state.Balances[babayaga]

		<-ctx.Done()

		endAndrejBalance := n.state.Balances[andrej]
		endBabayagaBalance := n.state.Balances[babayaga]

		expectedEndAndrejBalance := startingAndrejBalance - tx1.Value - tx2.Value + database.BlockReward
		expectedEndBabayagaBalance := startingBabayagaBalance + tx1.Value + tx2.Value + database.BlockReward

		if endAndrejBalance != expectedEndAndrejBalance {
			t.Fatalf("Andrej expected end balance is %d not %d", expectedEndAndrejBalance, endAndrejBalance)
		}

		if endBabayagaBalance != expectedEndBabayagaBalance {
			t.Fatalf("BabaYaga expected end balance is %d not %d", expectedEndBabayagaBalance, endBabayagaBalance)
		}

		t.Logf("Starting Andrej balance: %d", startingAndrejBalance)
		t.Logf("Starting BabaYaga balance: %d", startingBabayagaBalance)
		t.Logf("Ending Andrej balance: %d", endAndrejBalance)
		t.Logf("Ending BabaYaga balance: %d", endBabayagaBalance)
	}()

	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatal("was suppose to mine 1 pending TX into 1 valid blocks under 30m")
	}

	if len(n.pendingTXs) != 0 {
		t.Fatal("no pending TXs should be left to mine")
	}
}

func getTestDataDirPath() string {
	return filepath.Join(os.TempDir(), ".tbb_test")
}
