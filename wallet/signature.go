package wallet

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

func Sign(message []byte, privateKey *ecdsa.PrivateKey) (signature []byte, err error) {
	// hash the message to 32 bytes
	messageHash := sha256.Sum256(message)

	// sign the message using the private key
	signature, err = crypto.Sign(messageHash[:], privateKey)
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
	messageHash := sha256.Sum256(message)

	recoveredPubKey, err := crypto.SigToPub(messageHash[:], sig)
	if err != nil {
		return nil, fmt.Errorf("unable to verify message signature. %s", err.Error())
	}

	return recoveredPubKey, nil
}
