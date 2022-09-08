package node

import (
	"context"
	"fmt"
	"net/http"
	"the-blockchain-bar/database"
)

const (
	DefaultHTTPPort = 8080

	endpointStatus                = "/node/status"
	endpointSync                  = "/node/sync"
	endpointSyncQueryKeyFromBlock = "fromBlock"
)

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"isBootstrap"`
	IsActive    bool   `json:"isActive"`
}

type Node struct {
	dataDir    string
	port       uint64
	state      *database.State
	knownPeers map[string]PeerNode
}

func New(dataDir string, port uint64, bootstrap PeerNode) *Node {
	return &Node{
		dataDir: dataDir,
		port:    port,
		knownPeers: map[string]PeerNode{
			bootstrap.TcpAddress(): bootstrap,
		},
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, isActive bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, isActive}
}

func (p PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

func (n *Node) Run() error {
	ctx := context.Background()
	fmt.Println(fmt.Sprintf("Listening on HTTP port: %d", DefaultHTTPPort))

	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}

	defer state.Close()

	n.state = state

	go n.sync(ctx)

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})

	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, state)
	})

	http.HandleFunc(endpointStatus, func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, n)
	})

	http.HandleFunc(endpointSync, func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", DefaultHTTPPort), nil)
}
