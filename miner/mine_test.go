package miner

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"the-blockchain-bar/database"
	"the-blockchain-bar/resources"
	"the-blockchain-bar/wallet"
	"time"

	"github.com/test-go/testify/assert"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
)

func TestMine(t *testing.T) {
	minerPrivateKey, _, miner, err := generateKey()
	assert.NoError(t, err)

	pendingBlock, err := createRandomPendingBlock(minerPrivateKey, miner)
	assert.NoError(t, err)

	ctx := context.Background()

	minedBlock, err := Mine(ctx, pendingBlock)
	assert.NoError(t, err)

	minedBlockHash, err := minedBlock.Hash()
	assert.NoError(t, err)

	if !database.IsBlockHashValid(minedBlockHash) {
		t.Fatalf("invalid block hash: %s", minedBlockHash.Hex())
	}

	if minedBlock.Header.Miner.String() != miner.String() {
		t.Fatal("mined block miner should equal miner from pending block")
	}
}

func TestMineWithTimeout(t *testing.T) {
	minerPrivateKey, _, miner, err := generateKey()
	assert.NoError(t, err)

	pendingBlock, err := createRandomPendingBlock(minerPrivateKey, miner)
	assert.NoError(t, err)

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	if _, err := Mine(ctx, pendingBlock); err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock(privateKey *ecdsa.PrivateKey, minerAccount common.Address) (PendingBlock, error) {
	tx := database.NewTx(minerAccount, database.NewAccount(resources.TestKsBabaYagaAccount), 1, "")
	signedTx, err := wallet.SignTx(tx, privateKey)
	if err != nil {
		return PendingBlock{}, err
	}

	return NewPendingBlock(
		database.Hash{},
		0,
		minerAccount,
		[]database.SignedTx{signedTx},
	), nil
}

func generateKey() (*ecdsa.PrivateKey, ecdsa.PublicKey, common.Address, error) {
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, ecdsa.PublicKey{}, common.Address{}, err
	}

	pubKey := privateKey.PublicKey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	pubKeyBytesHash := crypto.Keccak256(pubKeyBytes[1:])

	account := common.BytesToAddress(pubKeyBytesHash[12:])

	return privateKey, pubKey, account, nil
}
