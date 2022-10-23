package node

import (
	"context"
	"fmt"
	"the-blockchain-bar/database"
	"the-blockchain-bar/miner"
	"time"
)

func (n *Node) mine(ctx context.Context) {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true
					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					if err := n.minePendingTXs(miningCtx); err != nil {
						fmt.Printf("an error occurred while mining pending transactions: %s\n", err.Error())
					}

					n.isMining = false
				}
			}()
		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nanother peer mined the next block '%s' faster :(\n", blockHash.Hex())

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := miner.NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Account,
		n.getPendingTXsAsArray(),
	)

	minedBlock, err := miner.Mine(ctx, blockToMine)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)

	if _, err := n.state.AddBlock(minedBlock); err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTXs(block database.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("removing the just-mined block TXs from the node in-memory pending TXs pool")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())

			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

func (n *Node) getPendingTXsAsArray() []database.SignedTx {
	txs := make([]database.SignedTx, len(n.pendingTXs))

	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}

	return txs
}
