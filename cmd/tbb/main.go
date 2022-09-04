package main

import (
	"errors"

	"github.com/spf13/cobra"
)

const flagDataDir = "datadir"

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
