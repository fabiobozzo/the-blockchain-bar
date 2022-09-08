package node

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(45 * time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Println("searching for new peers and locks...")

			n.fetchNewBlocksAndPeers()
		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) fetchNewBlocksAndPeers() {
	for _, knownPeer := range n.knownPeers {
		status, err := queryPeerStatus(knownPeer)
		if err != nil {
			fmt.Println("cannot query peer status: ", err)

			continue
		}

		localBlockNumber := n.state.LatestBlock().Header.Number
		if localBlockNumber < status.Number {
			newBlocksCount := status.Number - localBlockNumber

			fmt.Printf("found %d new blocks from peer %s\n", newBlocksCount, knownPeer.IP)
		}

		for _, maybeNewPeer := range status.KnownPeers {
			if _, isKnownPeer := n.knownPeers[maybeNewPeer.TcpAddress()]; !isKnownPeer {
				fmt.Printf("found new peer %s\n", maybeNewPeer.TcpAddress())
				n.knownPeers[maybeNewPeer.TcpAddress()] = maybeNewPeer
			}
		}
	}
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
