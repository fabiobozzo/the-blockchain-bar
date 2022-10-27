package database

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
)

func NewAccount(value string) common.Address {
	return common.HexToAddress(value)
}

type Tx struct {
	From  common.Address `json:"from"`
	To    common.Address `json:"to"`
	Value uint           `json:"value"`
	Nonce uint           `json:"nonce"`
	Data  string         `json:"data"`
	Time  uint64         `json:"time"`

	Gas      uint `json:"gas"`
	GasPrice uint `json:"gasPrice"`
}

type SignedTx struct {
	Tx
	Sig []byte `json:"signature"`
}

func NewTx(from, to common.Address, value, nonce, gas, gasPrice uint, data string) Tx {
	return Tx{
		from,
		to,
		value,
		nonce,
		data,
		uint64(time.Now().Unix()),
		gas,
		gasPrice,
	}
}

func NewBaseTx(from, to common.Address, value, nonce uint, data string) Tx {
	return NewTx(from, to, value, nonce, TxGas, TxGasPriceDefault, data)
}

func NewSignedTx(tx Tx, sig []byte) SignedTx {
	return SignedTx{tx, sig}
}

func (t Tx) IsReward() bool {
	return t.Data == "reward"
}

func (t Tx) Encode() ([]byte, error) {
	return json.Marshal(t)
}

func (t Tx) Hash() (Hash, error) {
	txJson, err := t.Encode()
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(txJson), nil
}

func (t Tx) Cost(isTip1Fork bool) uint {
	if isTip1Fork {
		return t.Value + t.GasCost()
	}

	return t.Value + TxFee
}

func (t Tx) GasCost() uint {
	return t.Gas * t.GasPrice
}

func (t SignedTx) Hash() (Hash, error) {
	txJson, err := t.Encode()
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(txJson), nil
}

func (t SignedTx) IsAuthentic() (bool, error) {
	// Convert to 32 bytes hash
	txHash, err := t.Tx.Hash()
	if err != nil {
		return false, err
	}

	// Recover the pub key from the tx signature and convert it to an Account
	recoveredPubKey, err := crypto.SigToPub(txHash[:], t.Sig)
	if err != nil {
		return false, err
	}

	recoveredPubKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPubKey.X, recoveredPubKey.Y)
	recoveredPubKeyBytesHash := crypto.Keccak256(recoveredPubKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPubKeyBytesHash[12:])

	return recoveredAccount.Hex() == t.Tx.From.Hex(), nil
}
