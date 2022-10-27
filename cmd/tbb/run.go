package main

import (
	"context"
	"fmt"
	"os"
	"the-blockchain-bar/database"
	"the-blockchain-bar/node"

	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Launches the TBB node and its HTTP API.",
		Run: func(cmd *cobra.Command, args []string) {
			sslEmail, _ := cmd.Flags().GetString(flagSSLEmail)
			isSSLDisabled, _ := cmd.Flags().GetBool(flagDisableSSL)
			miner, _ := cmd.Flags().GetString(flagMiner)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			bootstrapIp, _ := cmd.Flags().GetString(flagBootstrapIp)
			bootstrapPort, _ := cmd.Flags().GetUint64(flagBootstrapPort)
			bootstrapAcc, _ := cmd.Flags().GetString(flagBootstrapAcc)

			fmt.Println("Launching TBB node and its HTTP API...")

			bootstrap := node.NewPeerNode(
				bootstrapIp,
				bootstrapPort,
				true,
				database.NewAccount(bootstrapAcc),
				false,
				"",
			)

			if !isSSLDisabled {
				port = node.DefaultHTTPPort
			}

			version := fmt.Sprintf("%s.%s.%s-alpha %s %s", Major, Minor, Fix, shortGitCommit(GitCommit), Verbal)
			theNode := node.New(
				getDataDirFromCmd(cmd),
				ip,
				port,
				database.NewAccount(miner),
				bootstrap,
				version,
				node.DefaultMiningDifficulty,
			)

			if err := theNode.Run(context.Background(), isSSLDisabled, sslEmail); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)
	runCmd.Flags().Bool(flagDisableSSL, false, "should the HTTP API SSL certificate be disabled? (default false)")
	runCmd.Flags().String(flagSSLEmail, "", "your node's HTTP SSL certificate email")
	runCmd.Flags().String(flagMiner, node.DefaultMiner, "miner account of this node to receive block rewards")
	runCmd.Flags().String(flagIP, node.DefaultIP, "exposed IP for communication with peers")
	runCmd.Flags().Uint64(flagPort, node.HttpSSLPort, "exposed http port for communication with peers")
	runCmd.Flags().String(flagBootstrapIp, node.DefaultBootstrapIp, "default bootstrap server to interconnect peers")
	runCmd.Flags().Uint64(flagBootstrapPort, node.HttpSSLPort, "default bootstrap server port to interconnect peers")
	runCmd.Flags().String(flagBootstrapAcc, node.DefaultBootstrapAcc, "default bootstrap w/ 1M TBB tokens Genesis account")

	return runCmd
}
