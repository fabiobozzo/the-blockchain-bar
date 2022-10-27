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

const (
	nodeTestVersion             = "0.0.0-alpha-test"
	defaultTestMiningDifficulty = 2
)

func TestNode_Run(t *testing.T) {
	dataDir, err := getTestDataDirPath()
	assert.NoError(t, err)
	assert.NoError(t, utils.RemoveDir(dataDir))

	http.DefaultServeMux = new(http.ServeMux)

	n := New(dataDir, "127.0.0.1", 8085, database.NewAccount(DefaultMiner), PeerNode{}, nodeTestVersion, defaultTestMiningDifficulty)

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)

	if err := n.Run(ctx, true, ""); err != nil && err != http.ErrServerClosed {
		t.Fatalf("node server was suppose to close after 5s, instead: %s", err)
	}
}

func TestNode_Mining(t *testing.T) {
	dataDir, andrej, babayaga, err := setupTestNodeDir(1000000, 0)
	assert.NoError(t, err)
	defer utils.RemoveDir(dataDir)

	http.DefaultServeMux = new(http.ServeMux)

	// Required for AddPendingTX() to describe from what node the TX came from (local node in this case)
	localPeerNode := NewPeerNode("127.0.0.1", 8085, false, babayaga, true, nodeTestVersion)

	// Construct a new Node instance and configure Andrej as a miner
	node := New(dataDir, "127.0.0.1", 8085, andrej, PeerNode{}, nodeTestVersion, defaultTestMiningDifficulty)

	// Allow the mining to run for 30 minutes, in the worst case
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)

	// Schedule a new TX in 3 seconds from now, in a separate thread because the n.Run() few lines below is a blocking call
	go func() {
		time.Sleep(time.Second * 1)
		tx := database.NewBaseTx(andrej, babayaga, 1, 1, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)

			return
		}

		require.NoError(t, node.AddPendingTX(signedTx, localPeerNode))
	}()

	// Schedule a new TX in 12 seconds from now simulating that it came in - while the first TX is being mined
	go func() {
		time.Sleep(time.Second * 30)
		tx := database.NewBaseTx(andrej, babayaga, 2, 2, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)

			return
		}

		require.NoError(t, node.AddPendingTX(signedTx, localPeerNode))
	}()

	// Periodically check if we mined the 2 blocks
	go func() {
		ticker := time.NewTicker(10 * time.Second)

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

	// Run the node, mining and everything in a blocking call (hence the go-routines before)
	_ = node.Run(ctx, true, "")

	if node.state.LatestBlock().Header.Number != 1 {
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
	tc := []struct {
		name     string
		ForkTIP1 uint64
	}{
		{"Legacy", 35},  // Prior ForkTIP1 was activated on number 35
		{"ForkTIP1", 0}, // To test new blocks when the ForkTIP1 is active
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			babayaga := database.NewAccount(resources.TestKsBabaYagaAccount)
			andrej := database.NewAccount(resources.TestKsAndrejAccount)

			dataDir, err := getTestDataDirPath()
			if err != nil {
				t.Fatal(err)
			}

			genesisBalances := make(map[common.Address]uint)
			genesisBalances[andrej] = 1000000
			genesis := database.Genesis{Balances: genesisBalances, ForkTIP1: tc.ForkTIP1}
			genesisJson, err := json.Marshal(genesis)
			if err != nil {
				t.Fatal(err)
			}

			if err := database.InitDataDirIfNotExists(dataDir, genesisJson); err != nil {
				t.Fatal(err)
			}

			defer utils.RemoveDir(dataDir)

			if err := copyKeystoreFilesIntoTestDataDirPath(dataDir); err != nil {
				t.Fatal(err)
			}

			// Required for AddPendingTX() to describe
			// from what node the TX came from (local node in this case)
			nInfo := NewPeerNode(
				"127.0.0.1",
				8085,
				false,
				database.NewAccount(""),
				true,
				nodeTestVersion,
			)

			// Start mining with a high mining difficulty, just to be slow on purpose and let a synced block arrive first
			n := New(dataDir, nInfo.IP, nInfo.Port, babayaga, nInfo, nodeTestVersion, uint(5))

			// Allow the test to run for 30 minutes, in the worst case
			ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)

			tx1 := database.NewBaseTx(andrej, babayaga, 1, 1, "")
			tx2 := database.NewBaseTx(andrej, babayaga, 2, 2, "")

			if tc.name == "Legacy" {
				tx1.Gas = 0
				tx1.GasPrice = 0
				tx2.Gas = 0
				tx2.GasPrice = 0
			}

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

			// Pre-mine a valid block without running the `n.Run()`
			// with Andrej as a miner who will receive the block reward,
			// to simulate the block came on the fly from another peer
			validPreMinedPb := miner.NewPendingBlock(database.Hash{}, 0, andrej, []database.SignedTx{signedTx1})
			validSyncedBlock, err := miner.Mine(ctx, validPreMinedPb, defaultTestMiningDifficulty)
			if err != nil {
				t.Fatal(err)
			}

			// Add 2 new TXs into the BabaYaga's node, triggers mining
			go func() {
				time.Sleep(time.Second * (miningIntervalSeconds - 2))

				err := n.AddPendingTX(signedTx1, nInfo)
				if err != nil {
					t.Fatal(err)
				}

				err = n.AddPendingTX(signedTx2, nInfo)
				if err != nil {
					t.Fatal(err)
				}
			}()

			// Interrupt the previously started mining with a new synced block
			// BUT this block contains only 1 TX the previous mining activity tried to mine
			// which means the mining will start again for the one pending TX that is left and wasn't in
			// the synced block
			go func() {
				time.Sleep(time.Second * (miningIntervalSeconds + 2))
				if !n.isMining {
					t.Fatal("should be mining")
				}

				// Change the mining difficulty back to the testing level from previously purposefully slow, high value
				// Otherwise, the synced block would be invalid.
				n.ChangeMiningDifficulty(defaultTestMiningDifficulty)
				if _, err := n.state.AddBlock(validSyncedBlock); err != nil {
					t.Fatal(err)
				}

				// Mock the Andrej's block came from a network
				n.newSyncedBlocks <- validSyncedBlock

				time.Sleep(time.Second)
				if n.isMining {
					t.Fatal("synced block should have canceled mining")
				}

				// Mined TX1 by Andrej should be removed from the MemPool
				_, onlyTX2IsPending := n.pendingTXs[tx2Hash.Hex()]

				if len(n.pendingTXs) != 1 && !onlyTX2IsPending {
					t.Fatal("synced block should have canceled mining of already mined TX")
				}
			}()

			go func() {
				// Regularly check whenever both TXs are now mined
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

				// Take a snapshot of the DB balances
				// before the mining is finished and the 2 blocks
				// are created.
				startingAndrejBalance := n.state.Balances[andrej]
				startingBabayagaBalance := n.state.Balances[babayaga]

				// Wait until the 30 mins timeout is reached or
				// the 2 blocks got already mined and the closeNode() was triggered
				<-ctx.Done()

				endAndrejBalance := n.state.Balances[andrej]
				endBabayagaBalance := n.state.Balances[babayaga]

				// In TX1 Andrej transferred 1 TBB token to BabaYaga
				// In TX2 Andrej transferred 2 TBB tokens to BabaYaga

				var expectedEndAndrejBalance uint
				var expectedEndBabayagaBalance uint

				// Andrej will occur the cost of SENDING 2 TXs but will collect the reward for mining one block with tx1 in it
				// Babayaga will RECEIVE value from 2 TXs and will also collect the reward for mining one block with tx2 in it

				if n.state.IsTIP1Fork() {
					expectedEndAndrejBalance = startingAndrejBalance - tx1.Cost(true) - tx2.Cost(true) + database.BlockReward + tx1.GasCost()
					expectedEndBabayagaBalance = startingBabayagaBalance + tx1.Value + tx2.Value + database.BlockReward + tx2.GasCost()
				} else {
					expectedEndAndrejBalance = startingAndrejBalance - tx1.Cost(false) - tx2.Cost(false) + database.BlockReward + database.TxFee
					expectedEndBabayagaBalance = startingBabayagaBalance + tx1.Value + tx2.Value + database.BlockReward + database.TxFee
				}

				if endAndrejBalance != expectedEndAndrejBalance {
					t.Errorf("Andrej expected end balance is %d not %d", expectedEndAndrejBalance, endAndrejBalance)
				}

				if endBabayagaBalance != expectedEndBabayagaBalance {
					t.Errorf("BabaYaga expected end balance is %d not %d", expectedEndBabayagaBalance, endBabayagaBalance)
				}

				t.Logf("starting Andrej balance: %d", startingAndrejBalance)
				t.Logf("starting BabaYaga balance: %d", startingBabayagaBalance)
				t.Logf("ending Andrej balance: %d", endAndrejBalance)
				t.Logf("ending BabaYaga balance: %d", endBabayagaBalance)
			}()

			_ = n.Run(ctx, true, "")

			if n.state.LatestBlock().Header.Number != 1 {
				t.Fatal("was suppose to mine 2 pending TX into 2 valid blocks under 30m")
			}

			if len(n.pendingTXs) != 0 {
				t.Fatal("no pending TXs should be left to mine")
			}
		})
	}
}

func TestNode_ForgedTx(t *testing.T) {
	dataDir, andrej, babayaga, err := setupTestNodeDir(1000000, 0)
	assert.NoError(t, err)

	defer utils.RemoveDir(dataDir)

	n := New(dataDir, "127.0.0.1", 8085, andrej, PeerNode{}, nodeTestVersion, defaultTestMiningDifficulty)
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)
	andrejPeerNode := NewPeerNode("127.0.0.1", 8085, false, andrej, true, nodeTestVersion)

	txValue := uint(5)
	txNonce := uint(1)
	tx := database.NewBaseTx(andrej, babayaga, txValue, txNonce, "")

	validSignedTx, err := wallet.SignTxWithKeystoreAccount(tx, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Fatal(err)

		return
	}

	_ = n.AddPendingTX(validSignedTx, andrejPeerNode)

	go func() {
		ticker := time.NewTicker(time.Second * (miningIntervalSeconds - 3))
		wasForgedTxAdded := false

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					if wasForgedTxAdded && !n.isMining {
						closeNode()
						return
					}

					if !wasForgedTxAdded {
						// Attempt to forge the same TX but with modified time
						// Because the TX.time changed, the TX.signature will be considered forged
						// database.NewTx() changes the TX time
						forgedTx := database.NewBaseTx(andrej, babayaga, txValue, txNonce, "")
						// Use the signature from a valid TX
						forgedSignedTx := database.NewSignedTx(forgedTx, validSignedTx.Sig)

						_ = n.AddPendingTX(forgedSignedTx, andrejPeerNode)
						wasForgedTxAdded = true

						time.Sleep(time.Second * (miningIntervalSeconds + 3))
					}
				}
			}
		}
	}()

	_ = n.Run(ctx, true, "")

	if n.state.LatestBlock().Header.Number != 0 {
		t.Fatal("only one tx was supposed to be mined. the second tx was forged")
	}

	if n.state.Balances[babayaga] != txValue {
		t.Fatal("forged tx succeeded")
	}
}

func TestNode_ReplayedTx(t *testing.T) {
	dataDir, andrej, babayaga, err := setupTestNodeDir(1000000, 0)
	if err != nil {
		t.Error(err)
	}
	defer utils.RemoveDir(dataDir)

	n := New(dataDir, "127.0.0.1", 8085, andrej, PeerNode{}, nodeTestVersion, defaultTestMiningDifficulty)
	ctx, closeNode := context.WithCancel(context.Background())
	andrejPeerNode := NewPeerNode("127.0.0.1", 8085, false, andrej, true, nodeTestVersion)
	babayagaPeerNode := NewPeerNode("127.0.0.1", 8086, false, babayaga, true, nodeTestVersion)

	txValue := uint(5)
	txNonce := uint(1)
	tx := database.NewBaseTx(andrej, babayaga, txValue, txNonce, "")

	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Fatal(err)

		return
	}

	assert.NoError(t, n.AddPendingTX(signedTx, andrejPeerNode))

	go func() {
		ticker := time.NewTicker(time.Second * (miningIntervalSeconds - 3))
		wasReplayedTxAdded := false

		for {
			select {
			case <-ticker.C:
				// The Andrej's original TX got mined.
				// Execute the attack by replaying the TX again!
				if n.state.LatestBlock().Header.Number == 0 {
					if wasReplayedTxAdded && !n.isMining {
						closeNode()

						return
					}

					if !wasReplayedTxAdded {
						// Simulate the TX was submitted to different node
						n.archivedTXs = make(map[string]database.SignedTx)

						// Execute the attack
						_ = n.AddPendingTX(signedTx, babayagaPeerNode)
						wasReplayedTxAdded = true

						time.Sleep(time.Second * (miningIntervalSeconds + 3))
					}

				}
			}
		}
	}()

	_ = n.Run(ctx, true, "")

	if n.state.Balances[babayaga] != txValue {
		t.Fatalf("replayed attack was successful. babayaga balance is:%d should be:%d", n.state.Balances[babayaga], txValue)
	}

	if n.state.LatestBlock().Header.Number == 1 {
		t.Fatal("the second block was not suppose to be persisted because it contained a malicious tx")
	}
}

func TestNode_MiningSpamTransactions(t *testing.T) {
	tc := []struct {
		name     string
		ForkTIP1 uint64
	}{
		{"Legacy", 35},  // Prior ForkTIP1 was activated on number 35
		{"ForkTIP1", 0}, // To test new blocks when the ForkTIP1 is active
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {

			andrejBalance := uint(1000)
			babayagaBalance := uint(0)
			minerBalance := uint(0)
			minerKey, err := wallet.NewRandomKey()
			if err != nil {
				t.Fatal(err)
			}
			miner := minerKey.Address
			dataDir, andrej, babayaga, err := setupTestNodeDir(andrejBalance, tc.ForkTIP1)
			if err != nil {
				t.Fatal(err)
			}
			defer utils.RemoveDir(dataDir)

			n := New(dataDir, "127.0.0.1", 8085, miner, PeerNode{}, nodeTestVersion, defaultTestMiningDifficulty)
			ctx, closeNode := context.WithCancel(context.Background())
			minerPeerNode := NewPeerNode("127.0.0.1", 8085, false, miner, true, nodeTestVersion)

			txValue := uint(200)
			txCount := uint(4)
			spamTXs := make([]database.SignedTx, txCount)

			go func() {
				// Wait for the node to run and initialize its state and other components
				time.Sleep(time.Second)

				now := uint64(time.Now().Unix())
				// Schedule 4 transfers from Andrej -> BabaYaga
				for i := uint(1); i <= txCount; i++ {
					txNonce := i
					tx := database.NewBaseTx(andrej, babayaga, txValue, txNonce, "")
					// Ensure every TX has a unique timestamp and the nonce 0 has oldest timestamp, nonce 1 younger timestamp etc
					tx.Time = now - uint64(txCount-i*100)

					if tc.name == "Legacy" {
						tx.Gas = 0
						tx.GasPrice = 0
					}

					signedTx, err := wallet.SignTxWithKeystoreAccount(tx, andrej, resources.TestKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
					if err != nil {
						t.Fatal(err)
					}

					spamTXs[i-1] = signedTx
				}

				// Collect pre-signed TXs to an array to make sure all 4 fit into a block within the mining interval,
				// otherwise slower machines can start mining after TX 3 or so, making the test fail on e.g: Github Actions.
				for _, tx := range spamTXs {
					_ = n.AddPendingTX(tx, minerPeerNode)
				}
			}()

			go func() {
				// Periodically check if we mined the block
				ticker := time.NewTicker(10 * time.Second)

				for {
					select {
					case <-ticker.C:
						if !n.state.LatestBlockHash().IsEmpty() {
							closeNode()
							return
						}
					}
				}
			}()

			// Run the node, mining and everything in a blocking call (hence the go-routines before)
			_ = n.Run(ctx, true, "")

			var expectedAndrejBalance uint
			var expectedBabayagaBalance uint
			var expectedMinerBalance uint

			// in nutshell: sender occurs tx.Cost(), receiver gains tx.Value() and miner collects tx.GasCost()
			if n.state.IsTIP1Fork() {
				expectedAndrejBalance = andrejBalance
				expectedMinerBalance = minerBalance + database.BlockReward

				for _, tx := range spamTXs {
					expectedAndrejBalance -= tx.Cost(true)
					expectedMinerBalance += tx.GasCost()
				}

				expectedBabayagaBalance = babayagaBalance + (txCount * txValue)
			} else {
				expectedAndrejBalance = andrejBalance - (txCount * txValue) - (txCount * database.TxFee)
				expectedBabayagaBalance = babayagaBalance + (txCount * txValue)
				expectedMinerBalance = minerBalance + database.BlockReward + (txCount * database.TxFee)
			}

			if n.state.Balances[andrej] != expectedAndrejBalance {
				t.Errorf("andrej balance is incorrect. expected: %d. got: %d", expectedAndrejBalance, n.state.Balances[andrej])
			}

			if n.state.Balances[babayaga] != expectedBabayagaBalance {
				t.Errorf("babaYaga balance is incorrect. expected: %d. got: %d", expectedBabayagaBalance, n.state.Balances[babayaga])
			}

			if n.state.Balances[miner] != expectedMinerBalance {
				t.Errorf("miner balance is incorrect. expected: %d. got: %d", expectedMinerBalance, n.state.Balances[miner])
			}

			t.Logf("andrej final balance: %d TBB", n.state.Balances[andrej])
			t.Logf("babayaga final balance: %d TBB", n.state.Balances[babayaga])
			t.Logf("miner final balance: %d TBB", n.state.Balances[miner])
		})
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

// setupTestNodeDir creates a default testing node directory with 2 keystore accounts
// Remember to remove the dir once test finishes: defer fs.RemoveDir(dataDir)
func setupTestNodeDir(andrejBalance uint, forkTip1 uint64) (dataDir string, andrej, babayaga common.Address, err error) {
	babayaga = database.NewAccount(resources.TestKsBabaYagaAccount)
	andrej = database.NewAccount(resources.TestKsAndrejAccount)

	dataDir, err = getTestDataDirPath()
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[andrej] = andrejBalance
	genesis := database.Genesis{Balances: genesisBalances, ForkTIP1: forkTip1}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	if err := database.InitDataDirIfNotExists(dataDir, genesisJson); err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	if err := copyKeystoreFilesIntoTestDataDirPath(dataDir); err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	return dataDir, andrej, babayaga, nil
}
