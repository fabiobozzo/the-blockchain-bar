package node

import (
	"fmt"
	"net/http"
	"strconv"
	"the-blockchain-bar/database"
)

func listBalancesHandler(w http.ResponseWriter, _ *http.Request, state *database.State) {
	writeSuccessfulResponse(w, balancesResponse{
		Hash:     state.LatestBlockHash(),
		Balances: state.Balances,
	})
}

func txAddHandler(w http.ResponseWriter, r *http.Request, n *Node) {
	req := txAddRequest{}
	if err := requestFromBody(r, &req); err != nil {
		writeErrorResponse(w, err)

		return
	}

	tx := database.NewTx(database.NewAccount(req.From), database.NewAccount(req.To), req.Value, req.Data)

	if err := n.AddPendingTX(tx, n.info); err != nil {
		writeErrorResponse(w, err)

		return
	}

	writeSuccessfulResponse(w, txAddResponse{Success: true})
}

func statusHandler(w http.ResponseWriter, _ *http.Request, n *Node) {
	res := statusResponse{
		Hash:       n.state.LatestBlockHash(),
		Number:     n.state.LatestBlock().Header.Number,
		KnownPeers: n.knownPeers,
		PendingTXs: n.getPendingTXsAsArray(),
	}

	writeSuccessfulResponse(w, res)
}

func syncHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	// hash after which new blocks have to be returned
	reqHash := r.URL.Query().Get(endpointSyncQueryKeyFromBlock)

	hash := database.Hash{}
	if err := hash.UnmarshalText([]byte(reqHash)); err != nil {
		writeErrorResponse(w, err)

		return
	}

	// read newer blocks from db
	blocks, err := database.GetBlocksAfter(hash, node.dataDir)
	if err != nil {
		writeErrorResponse(w, err)

		return
	}

	writeSuccessfulResponse(w, syncResponse{Blocks: blocks})
}

func addPeerHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	peerIP := r.URL.Query().Get(endpointAddPeerQueryKeyIP)
	peerPortRaw := r.URL.Query().Get(endpointAddPeerQueryKeyPort)
	minerRaw := r.URL.Query().Get(endpointAddPeerQueryKeyMiner)

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeSuccessfulResponse(w, addPeerResponse{
			Success: false,
			Error:   err.Error(),
		})

		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, database.NewAccount(minerRaw), true)
	node.AddPeer(peer)

	fmt.Printf("peer '%s' was added to known peers\n", peer.TcpAddress())

	writeSuccessfulResponse(w, addPeerResponse{true, ""})
}
