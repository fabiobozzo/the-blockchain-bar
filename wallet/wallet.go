package wallet

import (
	"crypto/ecdsa"
	"fmt"
	"path/filepath"
	"the-blockchain-bar/database"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/crypto"
)

const keystoreDirName = "keystore"
const AndrejAccount = "0x22ba1F80452E6220c7cc6ea2D1e3EEDDaC5F694A"
const BabaYagaAccount = "0x21973d33e048f5ce006fd7b41f51725c30e4b76b"
const CaesarAccount = "0x84470a31D271ea400f34e7A697F36bE0e866a716"

func GetKeystoreDirPath(dataDir string) string {
	return filepath.Join(dataDir, keystoreDirName)
}

func NewKeystoreAccount(dataDir, password string) (common.Address, error) {
	ks := keystore.NewKeyStore(GetKeystoreDirPath(dataDir), keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := ks.NewAccount(password)
	if err != nil {
		return common.Address{}, err
	}

	return account.Address, nil
}

func SignTxWithKeystoreAccount(tx database.Tx, acc common.Address, pwd string) {

}

func Sign(message []byte, privateKey *ecdsa.PrivateKey) (signature []byte, err error) {
	// hash the message to 32 bytes
	messageHash := crypto.Keccak256(message)

	// sign the message using the private key
	signature, err = crypto.Sign(messageHash, privateKey)
	if err != nil {
		return nil, err
	}

	// verify the signature length
	if len(signature) != crypto.SignatureLength {
		return nil, fmt.Errorf(
			"wrong size for signature: got %d, want %d",
			len(signature),
			crypto.SignatureLength,
		)
	}

	return signature, nil
}

func Verify(message, sig []byte) (*ecdsa.PublicKey, error) {
	messageHash := crypto.Keccak256(message)

	recoveredPubKey, err := crypto.SigToPub(messageHash, sig)
	if err != nil {
		return nil, fmt.Errorf("unable to verify message signature. %s", err.Error())
	}

	return recoveredPubKey, nil
}
