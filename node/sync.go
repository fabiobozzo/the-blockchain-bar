package node

import (
	"context"
	"fmt"
	"net/http"
	"the-blockchain-bar/database"
	"time"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
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
		if n.ip == peer.IP && n.port == peer.Port {
			continue
		}

		fmt.Printf("Searching for new Peers and their Blocks and Peers: '%s'\n", peer.TcpAddress())

		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("Peer '%s' was removed from KnownPeers\n", peer.TcpAddress())
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

		if err := n.syncKnownPeers(peer, status); err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
	}
}

func (n *Node) joinKnownPeers(peer PeerNode) error {
	if peer.connected {
		return nil
	}

	url := fmt.Sprintf(
		"http://%s%s?%s=%s&%s=%d",
		peer.TcpAddress(),
		endpointAddPeer,
		endpointAddPeerQueryKeyIP,
		n.ip,
		endpointAddPeerQueryKeyPort,
		n.port,
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
	if localBlockNumber < status.Number {
		newBlocksCount := status.Number - localBlockNumber

		fmt.Printf("Found %d new blocks from Peer %s\n", newBlocksCount, peer.TcpAddress())

		blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash())
		if err != nil {
			return err
		}

		if err := n.state.AddBlocks(blocks); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) syncKnownPeers(peer PeerNode, status statusResponse) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnownPeer(statusPeer) {
			fmt.Printf("found new peer: %s\n", statusPeer.TcpAddress())
			n.AddPeer(statusPeer)
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
