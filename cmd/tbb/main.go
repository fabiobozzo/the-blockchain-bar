package main

import (
	"errors"
	"the-blockchain-bar/utils"

	"github.com/spf13/cobra"
)

const (
	flagDataDir       = "datadir"
	flagPort          = "port"
	flagIP            = "ip"
	flagMiner         = "miner"
	flagBootstrapAcc  = "bootstrap-account"
	flagBootstrapIp   = "bootstrap-ip"
	flagBootstrapPort = "bootstrap-port"
)

var ErrIncorrectUsage = errors.New("incorrect usage of tbb command")

func main() {
	var tbbCmd = &cobra.Command{
		Use:   "tbb",
		Short: "The Blockchain Bar CLI",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	tbbCmd.AddCommand(versionCmd)
	tbbCmd.AddCommand(balancesCmd())
	tbbCmd.AddCommand(runCmd())
	tbbCmd.AddCommand(migrateCmd())
	tbbCmd.AddCommand(walletCmd())

	if err := tbbCmd.Execute(); err != nil {
		fatal(err)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(
		flagDataDir,
		"",
		"Absolute path where all data is stored",
	)

	cmd.MarkFlagRequired(flagDataDir)
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)

	return utils.ExpandPath(dataDir)
}
