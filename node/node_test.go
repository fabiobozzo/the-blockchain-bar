package node

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"the-blockchain-bar/miner"
	"the-blockchain-bar/resources"
	"the-blockchain-bar/wallet"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"github.com/test-go/testify/require"

	"the-blockchain-bar/database"
	"the-blockchain-bar/utils"
)

func TestNode_Run(t *testing.T) {
	dataDir, err := getTestDataDirPath()
	assert.NoError(t, err)
	assert.NoError(t, utils.RemoveDir(dataDir))

	http.DefaultServeMux = new(http.ServeMux)

	n := New(dataDir, "127.0.0.1", 8085, database.NewAccount(DefaultMiner), PeerNode{})

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)

	if err := n.Run(ctx); err != http.ErrServerClosed {
		t.Fatalf("node server was suppose to close after 5s, instead: %s", err)
	}
}

func TestNode_Mining(t *testing.T) {
	andrej := database.NewAccount(resources.TestKsAndrejAccount)
	babayaga := database.NewAccount(resources.TestKsBabaYagaAccount)

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[andrej] = 1000000
	genesis := database.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	assert.NoError(t, err)

	dataDir, err := getTestDataDirPath()
	assert.NoError(t, err)
	assert.NoError(t, database.InitDataDirIfNotExists(dataDir, genesisJson))
	defer utils.RemoveDir(dataDir)

	assert.NoError(t, copyKeystoreFilesIntoTestDataDirPath(dataDir))

	http.DefaultServeMux = new(http.ServeMux)

	localPeerNode := NewPeerNode("127.0.0.1", 8085, false, database.NewAccount(""), true)
	node := New(dataDir, "127.0.0.1", 8085, andrej, PeerNode{})

	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)

	go func() {
		time.Sleep(time.Second * 1)
		tx := database.NewTx(andrej, babayaga, 1, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)

			return
		}

		require.NoError(t, node.AddPendingTX(signedTx, localPeerNode))
	}()

	go func() {
		time.Sleep(time.Second * 30)
		tx := database.NewTx(andrej, babayaga, 2, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)

			return
		}

		require.NoError(t, node.AddPendingTX(signedTx, localPeerNode))
	}()

	go func() {
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ticker.C:
				if node.state.LatestBlock().Header.Number == 2 {
					closeNode()
					return
				}
			}
		}
	}()

	_ = node.Run(ctx)

	if node.state.LatestBlock().Header.Number != 2 {
		t.Fatal("was suppose to mine 2 pending tx into 2 valid blocks under 30m")
	}
}

// The test logic summary:
//	- Babayaga runs the node
//  - Babayaga tries to mine 2 TXs
//  	- The mining gets interrupted because a new block from Andrej gets synced
//		- Andrej will get the block reward for this synced block
//		- The synced block contains 1 of the TXs Babayaga tried to mine
//	- Babayaga tries to mine 1 TX left
//		- Babayaga succeeds and gets her block reward
func TestNode_MiningStopsOnNewSyncedBlock(t *testing.T) {
	andrej := database.NewAccount(resources.TestKsAndrejAccount)
	babayaga := database.NewAccount(resources.TestKsBabaYagaAccount)

	dataDir, err := getTestDataDirPath()
	assert.NoError(t, err)

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[andrej] = 1000000
	genesis := database.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	assert.NoError(t, err)

	assert.NoError(t, database.InitDataDirIfNotExists(dataDir, genesisJson))
	defer utils.RemoveDir(dataDir)

	assert.NoError(t, copyKeystoreFilesIntoTestDataDirPath(dataDir))

	// Required for AddPendingTX() to describe from what node the TX came from (local node in this case)
	localPeerNode := NewPeerNode(
		"127.0.0.1",
		8085,
		false,
		database.NewAccount(""),
		true,
	)

	http.DefaultServeMux = new(http.ServeMux)
	node := New(dataDir, "127.0.0.1", 8085, babayaga, PeerNode{})
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)

	tx1 := database.Tx{From: andrej, To: babayaga, Value: 1, Time: 1579451695, Data: ""}
	tx2 := database.NewTx(andrej, babayaga, 2, "")

	signedTx1, err := wallet.SignTxWithKeystoreAccount(tx1, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}

	signedTx2, err := wallet.SignTxWithKeystoreAccount(tx2, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}

	tx2Hash, err := signedTx2.Hash()
	if err != nil {
		t.Error(err)
		return
	}

	validPreMinedPb := miner.NewPendingBlock(database.Hash{}, 0, andrej, []database.SignedTx{signedTx1})
	validSyncedBlock, err := miner.Mine(ctx, validPreMinedPb)
	if err != nil {
		t.Fatal(err)
	}

	// Add 2 new TXs into the Babayaga's node, triggers mining
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds - 2))

		if err := node.AddPendingTX(signedTx1, localPeerNode); err != nil {
			t.Fatal(err)
		}

		if err := node.AddPendingTX(signedTx2, localPeerNode); err != nil {
			t.Fatal(err)
		}
	}()

	// Interrupt the previously started mining with a new synced block.
	// BUT this block contains only 1 TX the previous mining activity tried to mine.
	// Which means the mining will start again for the one pending TX that is left and wasn't in the synced block.
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !node.isMining {
			t.Fatal("the node should be mining")
		}

		if _, err := node.state.AddBlock(validSyncedBlock); err != nil {
			t.Fatal(err)
		}

		// Mock the Andrej's block came from a network
		node.newSyncedBlocks <- validSyncedBlock

		time.Sleep(time.Second * 2)
		if node.isMining {
			t.Fatal("new received block should have canceled mining")
		}

		_, onlyTX2IsPending := node.pendingTXs[tx2Hash.Hex()]

		if len(node.pendingTXs) != 1 && !onlyTX2IsPending {
			t.Fatal("new received block should have canceled mining of already mined transaction")
		}

		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !node.isMining {
			t.Fatal("should be mining again the 1 TX not included in synced block")
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
				if node.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)

		// Take a snapshot of the DB balances before the mining is finished and the 2 blocks are created.
		startingAndrejBalance := node.state.Balances[andrej]
		startingBabayagaBalance := node.state.Balances[babayaga]

		<-ctx.Done()

		endAndrejBalance := node.state.Balances[andrej]
		endBabayagaBalance := node.state.Balances[babayaga]

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

	_ = node.Run(ctx)

	if node.state.LatestBlock().Header.Number != 1 {
		t.Fatal("was suppose to mine 1 pending TX into 1 valid blocks under 30m")
	}

	if len(node.pendingTXs) != 0 {
		t.Fatal("no pending TXs should be left to mine")
	}
}

// Creates dir like: "/tmp/tbb_test945924586"
func getTestDataDirPath() (string, error) {
	return ioutil.TempDir(os.TempDir(), "tbb_test")
}

func copyKeystoreFilesIntoTestDataDirPath(dataDir string) error {
	andrejKsPath := filepath.Join(utils.ProjectRootDir(), "resources", resources.TestKsAndrejFile)
	babayagaKsPath := filepath.Join(utils.ProjectRootDir(), "resources", resources.TestKsBabaYagaFile)

	andrejSrcKs, err := os.Open(andrejKsPath)
	if err != nil {
		return err
	}

	defer andrejSrcKs.Close()

	ksDir := filepath.Join(wallet.GetKeystoreDirPath(dataDir))
	if err := os.Mkdir(ksDir, 0777); err != nil {
		return err
	}

	andrejDstKs, err := os.Create(filepath.Join(ksDir, resources.TestKsAndrejFile))
	if err != nil {
		return err
	}

	defer andrejDstKs.Close()

	if _, err := io.Copy(andrejDstKs, andrejSrcKs); err != nil {
		return err
	}

	babayagaSrcKs, err := os.Open(babayagaKsPath)
	if err != nil {
		return err
	}

	defer babayagaSrcKs.Close()

	babayagaDstKs, err := os.Create(filepath.Join(ksDir, resources.TestKsBabaYagaFile))
	if err != nil {
		return err
	}

	defer babayagaDstKs.Close()

	if _, err := io.Copy(babayagaDstKs, babayagaSrcKs); err != nil {
		return err
	}

	return nil
}
