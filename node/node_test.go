package node

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
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

	n := New(dataDir, "127.0.0.1", 8085, PeerNode{})

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)

	if err := n.Run(ctx); err.Error() != http.ErrServerClosed.Error() {
		t.Fatalf("node server was suppose to close after 5s, instead: %s", err.Error())
	}
}

func TestNode_Mining(t *testing.T) {
	dataDir := getTestDataDirPath()
	if err := utils.RemoveDir(dataDir); err != nil {
		t.Fatal(err)
	}

	n := New(dataDir, "127.0.0.1", 8085, PeerNode{})
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)
	myselfNode := NewPeerNode("127.0.0.1", 8085, false, true)

	go func() {
		time.Sleep(time.Second * 1)
		tx := database.NewTx("andrej", "babayaga", 1, "")

		require.NoError(t, n.AddPendingTX(tx, myselfNode))
	}()

	go func() {
		time.Sleep(time.Second * 30)
		tx := database.NewTx("andrej", "babayaga", 2, "")

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
	dataDir := getTestDataDirPath()
	if err := utils.RemoveDir(dataDir); err != nil {
		t.Fatal(err)
	}

	n := New(dataDir, "127.0.0.1", 8085, PeerNode{})
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)

	tx := database.Tx{From: "andrej", To: "babayaga", Value: 1, Time: 1579451695, Data: ""}
	tx2 := database.NewTx("andrej", "babayaga", 2, "")
	tx2Hash, _ := tx2.Hash()

	validSyncedBlock := database.NewBlock(database.Hash{}, 1, 1453450257, 1579451704, []database.Tx{tx})

	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds - 2))

		myself := NewPeerNode("127.0.0.1", 8085, false, true)

		if err := n.AddPendingTX(tx, myself); err != nil {
			t.Fatal(err)
		}

		if err := n.AddPendingTX(tx2, myself); err != nil {
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
		t.Fatal("was suppose to mine 2 pending TX into 2 valid blocks under 30m")
	}

	if len(n.pendingTXs) != 0 {
		t.Fatal("no pending TXs should be left to mine")
	}
}

func getTestDataDirPath() string {
	return filepath.Join(os.TempDir(), ".tbb_test")
}
