package node

import (
	"encoding/json"
	"net/http"
	"the-blockchain-bar/database"
)

type errorResponse struct {
	Error string `json:"error"`
}

type balancesResponse struct {
	Hash     database.Hash             `json:"blockHash"`
	Balances map[database.Account]uint `json:"balances"`
}

type txAddResponse struct {
	Hash database.Hash `json:"blockHash"`
}

type statusResponse struct {
	Hash       database.Hash `json:"blockHash"`
	Number     uint64        `json:"blockNumber"`
	KnownPeers []PeerNode    `json:"peersKnown"`
}

func writeSuccessfulResponse(w http.ResponseWriter, content interface{}) {
	contentJson, err := json.Marshal(content)
	if err != nil {
		writeErrorResponse(w, err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(contentJson)
}

func writeErrorResponse(w http.ResponseWriter, err error) {
	jsonErrRes, _ := json.Marshal(errorResponse{err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonErrRes)
}
