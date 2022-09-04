package node

import "the-blockchain-bar/database"

type errorResponse struct {
	Error string `json:"error"`
}

type balancesResponse struct {
	Hash     database.Hash             `json:"blockHash"`
	Balances map[database.Account]uint `json:"balances"`
}

type txAddRequest struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint   `json:"value"`
	Data  string `json:"data"`
}

type txAddResponse struct {
	Hash database.Hash `json:"blockHash"`
}
