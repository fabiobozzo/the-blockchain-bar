package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignAndVerifySignature(t *testing.T) {
	// Generate Private Key on the fly
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Convert the Public Key to bytes with elliptic curve settings
	publicKey := privateKey.PublicKey
	publicKeyBytes := elliptic.Marshal(crypto.S256(), publicKey.X, publicKey.Y)

	// Hash the Public Key to 32 bytes
	publicKeyBytesHash := crypto.Keccak256(publicKeyBytes[1:])

	// The last 20 bytes of the Public Key hash will be its public username
	account := common.BytesToAddress(publicKeyBytesHash[12:])

	msg := []byte("the Web3Coach students are awesome")

	// Sign a message -> generate message's signature
	signature, err := Sign(msg, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	// Recover a Public Key from the signature
	recoveredPubKey, err := Verify(msg, signature)
	if err != nil {
		t.Fatal(err)
	}

	// Convert the Public Key to username again
	recoveredPubKeyBytes := elliptic.Marshal(
		crypto.S256(),
		recoveredPubKey.X,
		recoveredPubKey.Y,
	)
	recoveredPubKeyBytesHash := crypto.Keccak256(recoveredPubKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPubKeyBytesHash[12:])

	// Compare the usernames match:
	// the pub key derived from the private key and the one recovered from the signed message do match
	if account.Hex() != recoveredAccount.Hex() {
		t.Fatalf(
			"msg was signed by account %s but signature recovery produced an account %s",
			account.Hex(),
			recoveredAccount.Hex(),
		)
	}
}
