package miner

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"the-blockchain-bar/database"
	"the-blockchain-bar/resources"
	"the-blockchain-bar/wallet"
	"time"

	"github.com/test-go/testify/assert"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
)

const defaultTestMiningDifficulty = 2

func TestValidBlockHash(t *testing.T) {
	hexHash := "0000fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa"
	var hash = database.Hash{}

	hex.Decode(hash[:], []byte(hexHash))

	isValid := database.IsBlockHashValid(hash, defaultTestMiningDifficulty)
	if !isValid {
		t.Fatalf("hash '%s' starting with 4 zeroes is suppose to be valid", hexHash)
	}
}

func TestInvalidBlockHash(t *testing.T) {
	hexHash := "0001fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa"
	var hash = database.Hash{}

	hex.Decode(hash[:], []byte(hexHash))

	isValid := database.IsBlockHashValid(hash, defaultTestMiningDifficulty)
	if isValid {
		t.Fatal("hash is not suppose to be valid")
	}
}

func TestMine(t *testing.T) {
	minerPrivateKey, _, miner, err := generateKey()
	assert.NoError(t, err)

	pendingBlock, err := createRandomPendingBlock(minerPrivateKey, miner)
	assert.NoError(t, err)

	ctx := context.Background()

	minedBlock, err := Mine(ctx, pendingBlock, defaultTestMiningDifficulty)
	assert.NoError(t, err)

	minedBlockHash, err := minedBlock.Hash()
	assert.NoError(t, err)

	if !database.IsBlockHashValid(minedBlockHash, defaultTestMiningDifficulty) {
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

	if _, err := Mine(ctx, pendingBlock, defaultTestMiningDifficulty); err == nil {
		t.Fatal(err)
	}
}

func createRandomPendingBlock(privateKey *ecdsa.PrivateKey, minerAccount common.Address) (PendingBlock, error) {
	tx := database.NewBaseTx(minerAccount, database.NewAccount(resources.TestKsBabaYagaAccount), 1, 1, "")
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
