package main

import (
	"context"
	"fmt"
	"the-blockchain-bar/database"
	"the-blockchain-bar/node"
	"the-blockchain-bar/wallet"
	"time"

	"github.com/spf13/cobra"
)

var migrateCmd = func() *cobra.Command {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrates the blockchain database according to new business rules.",
		Run: func(cmd *cobra.Command, args []string) {
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)

			andrej := database.NewAccount(wallet.AndrejAccount)
			babayaga := database.NewAccount(wallet.BabaYagaAccount)
			caesar := database.NewAccount(wallet.CaesarAccount)

			peer := node.NewPeerNode(
				"127.0.0.1",
				8080,
				true,
				andrej,
				false,
			)

			n := node.New(getDataDirFromCmd(cmd), ip, port, database.NewAccount(miner), peer)

			n.AddPendingTX(database.NewTx(andrej, andrej, 3, ""), peer)
			n.AddPendingTX(database.NewTx(andrej, babayaga, 2000, ""), peer)
			n.AddPendingTX(database.NewTx(babayaga, andrej, 1, ""), peer)
			n.AddPendingTX(database.NewTx(babayaga, caesar, 1000, ""), peer)
			n.AddPendingTX(database.NewTx(babayaga, andrej, 50, ""), peer)

			ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)
			go func() {
				ticker := time.NewTicker(time.Second * 10)

				for {
					select {
					case <-ticker.C:
						if !n.LatestBlockHash().IsEmpty() {
							closeNode()
							return
						}
					}
				}
			}()

			if err := n.Run(ctx); err != nil {
				fmt.Println("error while migrating transactions: ", err)
			}
		},
	}

	addDefaultRequiredFlags(migrateCmd)

	migrateCmd.Flags().String(flagMiner, node.DefaultMiner, "miner account of this node to receive block rewards")
	migrateCmd.Flags().String(flagIP, node.DefaultIP, "exposed IP for communication with peers")
	migrateCmd.Flags().Uint64(flagPort, node.DefaultHTTPPort, "exposed HTTP port for communication with peers")

	return migrateCmd
}
