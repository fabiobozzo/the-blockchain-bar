package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	Hash       database.Hash       `json:"blockHash"`
	Number     uint64              `json:"blockNumber"`
	KnownPeers map[string]PeerNode `json:"peersKnown"`
}

type syncResponse struct {
	Blocks []database.Block `json:"blocks"`
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

func readResponse(r *http.Response, reqBody interface{}) error {
	reqBodyJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body. %s", err.Error())
	}
	defer r.Body.Close()

	err = json.Unmarshal(reqBodyJson, reqBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal response body. %s", err.Error())
	}

	return nil
}