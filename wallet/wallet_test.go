package wallet

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"the-blockchain-bar/database"
	"the-blockchain-bar/utils"

	"github.com/test-go/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/davecgh/go-spew/spew"
)

// The password for testing keystore files:
// 	./resources/test_andrej--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57
// 	./resources/test_babayaga--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8
const testKeystoreAccountsPwd = "security123"

func TestSignTxWithKeystoreAccount(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "wallet_test")
	assert.NoError(t, err)

	defer utils.RemoveDir(tmpDir)

	andrej, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	assert.NoError(t, err)

	babayaga, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	assert.NoError(t, err)

	tx := database.NewBaseTx(andrej, babayaga, 100, 1, "")

	signedTx, err := SignTxWithKeystoreAccount(tx, andrej, testKeystoreAccountsPwd, GetKeystoreDirPath(tmpDir))
	assert.NoError(t, err)

	spew.Dump(signedTx.Encode())

	ok, err := signedTx.IsAuthentic()
	assert.NoError(t, err)

	if !ok {
		t.Fatal("the tx was signed by 'from' account and should have been authentic")

		return
	}

	// Test marshaling
	signedTxJson, err := json.Marshal(signedTx)
	if err != nil {
		t.Error(err)

		return
	}

	var signedTxUnmarshalled database.SignedTx
	if err = json.Unmarshal(signedTxJson, &signedTxUnmarshalled); err != nil {
		t.Error(err)

		return
	}

	require.Equal(t, signedTx, signedTxUnmarshalled)
}

func TestSignForgedTxWithKeystoreAccount(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "wallet_test")
	assert.NoError(t, err)

	defer utils.RemoveDir(tmpDir)

	hacker, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	assert.NoError(t, err)

	babayaga, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	assert.NoError(t, err)

	forgedTx := database.NewBaseTx(babayaga, hacker, 100, 1, "")

	signedTx, err := SignTxWithKeystoreAccount(forgedTx, hacker, testKeystoreAccountsPwd, GetKeystoreDirPath(tmpDir))
	assert.NoError(t, err)

	ok, err := signedTx.IsAuthentic()
	assert.NoError(t, err)

	if ok {
		t.Fatal("the TX 'from' attribute was forged and should have not be authentic")
	}
}
