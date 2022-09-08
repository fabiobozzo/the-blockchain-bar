package main

import (
	"fmt"
	"os"
	"the-blockchain-bar/node"

	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Launches the TBB node and its HTTP API.",
		Run: func(cmd *cobra.Command, args []string) {
			port, _ := cmd.Flags().GetUint64(flagPort)

			fmt.Println("Launching TBB node and its HTTP API...")

			bootstrap := node.NewPeerNode("18.184.213.146", 8080, true, true)
			theNode := node.New(getDataDirFromCmd(cmd), port, bootstrap)

			if err := theNode.Run(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)
	runCmd.Flags().Uint64(flagPort, node.DefaultHTTPPort, "exposed http port for communication with peers")

	return runCmd
}
