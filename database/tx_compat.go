package database

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
)

// MarshalJSON is the main source of truth for encoding a TX for hash calculation from expected attributes.
//
// The logic is a bit ugly and hacky but prevents infinite marshaling loops of embedded objects
// and allows the structure to change with new TIPs.
func (t Tx) MarshalJSON() ([]byte, error) {
	// Prior TIP1
	if t.Gas == 0 {
		return json.Marshal(struct {
			From  common.Address `json:"from"`
			To    common.Address `json:"to"`
			Value uint           `json:"value"`
			Nonce uint           `json:"nonce"`
			Data  string         `json:"data"`
			Time  uint64         `json:"time"`
		}{
			From:  t.From,
			To:    t.To,
			Value: t.Value,
			Nonce: t.Nonce,
			Data:  t.Data,
			Time:  t.Time,
		})
	}

	// TIP1 tx format (w/ Gas)
	return json.Marshal(struct {
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		Gas      uint           `json:"gas"`
		GasPrice uint           `json:"gasPrice"`
		Value    uint           `json:"value"`
		Nonce    uint           `json:"nonce"`
		Data     string         `json:"data"`
		Time     uint64         `json:"time"`
	}{
		From:     t.From,
		To:       t.To,
		Gas:      t.Gas,
		GasPrice: t.GasPrice,
		Value:    t.Value,
		Nonce:    t.Nonce,
		Data:     t.Data,
		Time:     t.Time,
	})
}

func (t SignedTx) MarshalJSON() ([]byte, error) {
	// Prior TIP1
	if t.Gas == 0 {
		return json.Marshal(struct {
			From  common.Address `json:"from"`
			To    common.Address `json:"to"`
			Value uint           `json:"value"`
			Nonce uint           `json:"nonce"`
			Data  string         `json:"data"`
			Time  uint64         `json:"time"`
			Sig   []byte         `json:"signature"`
		}{
			From:  t.From,
			To:    t.To,
			Value: t.Value,
			Nonce: t.Nonce,
			Data:  t.Data,
			Time:  t.Time,
			Sig:   t.Sig,
		})
	}

	// TIP1 tx format (w/ Gas)
	return json.Marshal(struct {
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		Gas      uint           `json:"gas"`
		GasPrice uint           `json:"gasPrice"`
		Value    uint           `json:"value"`
		Nonce    uint           `json:"nonce"`
		Data     string         `json:"data"`
		Time     uint64         `json:"time"`
		Sig      []byte         `json:"signature"`
	}{
		From:     t.From,
		To:       t.To,
		Gas:      t.Gas,
		GasPrice: t.GasPrice,
		Value:    t.Value,
		Nonce:    t.Nonce,
		Data:     t.Data,
		Time:     t.Time,
		Sig:      t.Sig,
	})
}
