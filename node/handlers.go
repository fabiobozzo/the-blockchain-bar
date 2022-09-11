package node

import (
	"fmt"
	"net/http"
	"strconv"
	"the-blockchain-bar/database"
	"time"
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

	block := database.NewBlock(
		state.LatestBlockHash(),
		state.LatestBlock().Header.Number+1,
		uint64(time.Now().Unix()),
		[]database.Tx{tx},
	)

	hash, err := state.AddBlock(block)
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

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeSuccessfulResponse(w, addPeerResponse{
			Success: false,
			Error:   err.Error(),
		})

		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, true)
	node.AddPeer(peer)

	fmt.Printf("Peer '%s' was added into KnownPeers\n", peer.TcpAddress())

	writeSuccessfulResponse(w, addPeerResponse{true, ""})
}
