package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"the-blockchain-bar/database"

	"github.com/caddyserver/certmagic"
	"github.com/ethereum/go-ethereum/common"
)

const (
	DefaultIP               = "127.0.0.1"
	DefaultHTTPPort         = 8080
	DefaultBootstrapIp      = "node.tbb.web3.coach"
	DefaultBootstrapPort    = 8080
	DefaultBootstrapAcc     = "0x09ee50f2f37fcba1845de6fe5c762e83e65e755c"
	DefaultMiner            = "0x0000000000000000000000000000000000000000"
	DefaultMiningDifficulty = 3
	HttpSSLPort             = 443

	endpointBalances = "/balances/list"
	endpointStatus   = "/node/status"
	endpointAddTx    = "/tx/add"

	endpointSync                  = "/node/sync"
	endpointSyncQueryKeyFromBlock = "fromBlock"

	endpointAddPeer                = "/node/peer"
	endpointAddPeerQueryKeyIP      = "ip"
	endpointAddPeerQueryKeyPort    = "port"
	endpointAddPeerQueryKeyMiner   = "miner"
	endpointAddPeerQueryKeyVersion = "version"

	miningIntervalSeconds = 10
)

type PeerNode struct {
	NodeVersion string         `json:"node_version"`
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	IsBootstrap bool           `json:"isBootstrap"`
	Account     common.Address `json:"account"`

	// Whenever my node already established connection, sync with this Peer
	connected bool
}

type Node struct {
	nodeVersion      string
	dataDir          string
	info             PeerNode
	state            *database.State
	knownPeers       map[string]PeerNode
	pendingTXs       map[string]database.SignedTx
	archivedTXs      map[string]database.SignedTx
	newSyncedBlocks  chan database.Block
	newPendingTXs    chan database.SignedTx
	isMining         bool
	miningDifficulty uint // number of zeroes the hash must start with to be considered valid. default: 3
}

func New(dataDir string, ip string, port uint64, account common.Address, bootstrap PeerNode, version string, miningDifficulty uint) *Node {
	knownPeers := make(map[string]PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap

	return &Node{
		dataDir:          dataDir,
		info:             NewPeerNode(ip, port, false, account, true, version),
		knownPeers:       knownPeers,
		pendingTXs:       make(map[string]database.SignedTx),
		archivedTXs:      make(map[string]database.SignedTx),
		newSyncedBlocks:  make(chan database.Block),
		newPendingTXs:    make(chan database.SignedTx, 10000),
		isMining:         false,
		miningDifficulty: miningDifficulty,
		nodeVersion:      version,
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, account common.Address, connected bool, version string) PeerNode {
	return PeerNode{
		version,
		ip,
		port,
		isBootstrap,
		account,
		connected,
	}
}

func (pn PeerNode) ApiProtocol() string {
	if pn.Port == HttpSSLPort {
		return "https"
	}

	return "http"
}

func (pn PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func (n *Node) Run(ctx context.Context, isSSLDisabled bool, sslEmail string) error {
	state, err := database.NewStateFromDisk(n.dataDir, n.miningDifficulty)
	if err != nil {
		return err
	}

	defer state.Close()

	n.state = state
	fmt.Println("blockchain state:")
	fmt.Printf("	- height: %d\n", n.state.LatestBlock().Header.Number)
	fmt.Printf("	- hash: %s\n", n.state.LatestBlockHash().Hex())

	go n.sync(ctx)
	go n.mine(ctx)

	return n.startHttpServer(ctx, isSSLDisabled, sslEmail)
}

func (n *Node) LatestBlockHash() database.Hash {
	return n.state.LatestBlockHash()
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
}

func (n *Node) AddPendingTX(signedTx database.SignedTx, fromPeer PeerNode) error {
	txHash, err := signedTx.Hash()
	if err != nil {
		return err
	}

	txJson, err := json.Marshal(signedTx)
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isAlreadyArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isAlreadyArchived {
		fmt.Printf("added Pending TX %s from peer %s\n", txJson, fromPeer.TcpAddress())
		n.pendingTXs[txHash.Hex()] = signedTx
		n.newPendingTXs <- signedTx
	}

	return nil
}

func (n *Node) ChangeMiningDifficulty(newDifficulty uint) {
	n.miningDifficulty = newDifficulty
	n.state.ChangeMiningDifficulty(newDifficulty)
}

func (n *Node) startHttpServer(ctx context.Context, isSSLDisabled bool, sslEmail string) error {
	router := http.NewServeMux()

	router.HandleFunc(endpointBalances, func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, n.state)
	})

	router.HandleFunc(endpointAddTx, func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, n)
	})

	router.HandleFunc(endpointStatus, func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, n)
	})

	router.HandleFunc(endpointSync, func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})

	router.HandleFunc(endpointAddPeer, func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	if isSSLDisabled {
		server := &http.Server{Addr: fmt.Sprintf(":%d", n.info.Port), Handler: router}

		go func() {
			<-ctx.Done()
			_ = server.Close()
		}()

		fmt.Println(fmt.Sprintf("Listening on %s:%d", n.info.IP, n.info.Port))

		// This shouldn't be an error!
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}

	} else {
		certmagic.DefaultACME.Email = sslEmail

		return certmagic.HTTPS([]string{n.info.IP}, router)
	}

	return nil
}
