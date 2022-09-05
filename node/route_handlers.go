package node

import (
	"net/http"
	"the-blockchain-bar/database"
)

func listBalancesHandler(w http.ResponseWriter, _ *http.Request, state *database.State) {
	writeSuccessfulResponse(w, balancesResponse{
		Hash:     state.LatestBlockHash(),
		Balances: state.Balances,
	})
}

func txAddHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	req := txAddRequest{}
	if err := requestFromBody(r, &req); err != nil {
		writeErrorResponse(w, err)

		return
	}

	tx := database.NewTx(database.NewAccount(req.From), database.NewAccount(req.To), req.Value, req.Data)

	if err := state.AddTx(tx); err != nil {
		writeErrorResponse(w, err)

		return
	}

	hash, err := state.Persist()
	if err != nil {
		writeErrorResponse(w, err)

		return
	}

	writeSuccessfulResponse(w, txAddResponse{hash})
}

func statusHandler(w http.ResponseWriter, r *http.Request, n *Node) {
	res := statusResponse{
		Hash:       n.state.LatestBlockHash(),
		Number:     n.state.LatestBlock().Header.Number,
		KnownPeers: n.knownPeers,
	}

	writeSuccessfulResponse(w, res)
}
