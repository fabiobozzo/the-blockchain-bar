package node

import (
	"fmt"
	"net/http"
	"strconv"
	"the-blockchain-bar/database"
	"the-blockchain-bar/wallet"

	"github.com/ethereum/go-ethereum/common"
)

func listBalancesHandler(w http.ResponseWriter, _ *http.Request, state *database.State) {
	writeSuccessfulResponse(w, balancesResponse{
		Hash:     state.LatestBlockHash(),
		Balances: state.Balances,
	})
}

func txAddHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := txAddRequest{}
	if err := requestFromBody(r, &req); err != nil {
		writeErrorResponse(w, err)

		return
	}

	from := database.NewAccount(req.From)
	to := database.NewAccount(req.To)

	if from.String() == common.HexToAddress("").String() {
		writeErrorResponse(w, fmt.Errorf("%s is an invalid 'from' sender", from.String()))

		return
	}

	if req.KeystorePassword == "" {
		writeErrorResponse(w, fmt.Errorf("password to decrypt the %s account is required. 'pwd' is empty", from.String()))

		return
	}

	// Build the unsigned transaction
	nonce := node.state.GetNextNonceByAccount(from)
	tx := database.NewTx(from, to, req.Value, nonce, req.Gas, req.GasPrice, req.Data)

	// Decrypt the Private key stored in Keystore file and Sign the TX
	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, from, req.KeystorePassword, wallet.GetKeystoreDirPath(node.dataDir))
	if err != nil {
		writeErrorResponse(w, err)

		return
	}

	// Add TX to the MemPool, ready to be mined
	if err := node.AddPendingTX(signedTx, node.info); err != nil {
		writeErrorResponse(w, err)

		return
	}

	writeSuccessfulResponse(w, txAddResponse{Success: true})
}

func statusHandler(w http.ResponseWriter, _ *http.Request, n *Node) {
	res := statusResponse{
		Hash:        n.state.LatestBlockHash(),
		Number:      n.state.LatestBlock().Header.Number,
		KnownPeers:  n.knownPeers,
		PendingTXs:  n.getPendingTXsAsArray(),
		NodeVersion: n.nodeVersion,
		Account:     database.NewAccount(n.info.Account.String()),
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
	versionRaw := r.URL.Query().Get(endpointAddPeerQueryKeyVersion)

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeSuccessfulResponse(w, addPeerResponse{
			Success: false,
			Error:   err.Error(),
		})

		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, database.NewAccount(minerRaw), true, versionRaw)
	node.AddPeer(peer)

	fmt.Printf("peer '%s' was added to known peers\n", peer.TcpAddress())

	writeSuccessfulResponse(w, addPeerResponse{true, ""})
}
