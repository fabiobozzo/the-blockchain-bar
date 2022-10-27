package node

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"the-blockchain-bar/database"
	"time"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			n.doSync()
		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doSync() {
	for _, peer := range n.knownPeers {
		if (n.info.IP == peer.IP && n.info.Port == peer.Port) || peer.IP == "" {
			continue
		}

		fmt.Printf("searching for new peers and their blocks and their known peers: '%s'\n", peer.TcpAddress())

		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("peer '%s' was removed from KnownPeers\n", peer.TcpAddress())
			n.RemovePeer(peer)

			continue
		}

		if err := n.joinKnownPeers(peer); err != nil {
			fmt.Printf("error joining known peers: %s\n", err)

			continue
		}

		if err = n.syncBlocks(peer, status); err != nil {
			fmt.Printf("error syncing new blocks: %s\n", err)

			continue
		}

		if err := n.syncKnownPeers(status); err != nil {
			fmt.Printf("error syncing new known peers: %s\n", err)

			continue
		}

		if err := n.syncPendingTXs(peer, status.PendingTXs); err != nil {
			fmt.Printf("error syncing new pending transactions: %s\n", err)

			continue
		}
	}
}

func (n *Node) joinKnownPeers(peer PeerNode) error {
	if peer.connected {
		return nil
	}

	url := fmt.Sprintf(
		"%s://%s%s?%s=%s&%s=%d&%s=%s&%s=%s",
		peer.ApiProtocol(),
		peer.TcpAddress(),
		endpointAddPeer,
		endpointAddPeerQueryKeyIP,
		n.info.IP,
		endpointAddPeerQueryKeyPort,
		n.info.Port,
		endpointAddPeerQueryKeyMiner,
		n.info.Account.String(),
		endpointAddPeerQueryKeyVersion,
		url.QueryEscape(n.info.NodeVersion),
	)

	rawResponse, err := http.Get(url)
	if err != nil {
		return err
	}

	res := addPeerResponse{}
	if err := readResponse(rawResponse, &res); err != nil {
		return err
	}

	if res.Error != "" {
		return fmt.Errorf("error adding local peer to remote peer: %s", res.Error)
	}

	knownPeer := n.knownPeers[peer.TcpAddress()]
	knownPeer.connected = res.Success

	if !res.Success {
		return fmt.Errorf("unable to join to %s peers", peer.TcpAddress())
	}

	return nil
}

func (n *Node) syncBlocks(peer PeerNode, status statusResponse) error {
	localBlockNumber := n.state.LatestBlock().Header.Number

	// If the peer has no blocks, ignore it
	if status.Hash.IsEmpty() {
		return nil
	}

	// If the peer has fewer blocks than us, ignore it
	if status.Number < localBlockNumber {
		return nil
	}

	// If it's the genesis block, and we already synced it, then ignore it
	if status.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	newBlocksCount := status.Number - localBlockNumber
	if localBlockNumber == 0 && status.Number == 0 { // Display found 1 new block if we sync the genesis block 0
		newBlocksCount = 1
	}
	fmt.Printf("found %d new blocks from peer %s\n", newBlocksCount, peer.TcpAddress())

	blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash())
	if err != nil {
		return err
	}

	for _, block := range blocks {
		if _, err = n.state.AddBlock(block); err != nil {
			return err
		}

		n.newSyncedBlocks <- block
	}

	return nil
}

func (n *Node) syncKnownPeers(status statusResponse) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnownPeer(statusPeer) {
			fmt.Printf("found new peer: %s\n", statusPeer.TcpAddress())
			n.AddPeer(statusPeer)
		}
	}

	return nil
}

func (n *Node) syncPendingTXs(peer PeerNode, txs []database.SignedTx) error {
	for _, tx := range txs {
		if err := n.AddPendingTX(tx, peer); err != nil {
			return err
		}
	}

	return nil
}

func queryPeerStatus(peer PeerNode) (statusResponse, error) {
	res, err := http.Get(fmt.Sprintf("http://%s/%s", peer.TcpAddress(), endpointStatus))
	if err != nil {
		return statusResponse{}, err
	}

	statusRes := statusResponse{}
	if err = readResponse(res, &statusRes); err != nil {
		return statusResponse{}, err
	}

	return statusRes, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock database.Hash) ([]database.Block, error) {
	fmt.Printf("Importing blocks from Peer %s...\n", peer.TcpAddress())

	url := fmt.Sprintf(
		"http://%s%s?%s=%s",
		peer.TcpAddress(),
		endpointSync,
		endpointSyncQueryKeyFromBlock,
		fromBlock.Hex(),
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	syncRes := syncResponse{}
	if err := readResponse(res, &syncRes); err != nil {
		return nil, err
	}

	return syncRes.Blocks, nil
}
