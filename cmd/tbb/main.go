package main

import (
	"errors"

	"github.com/spf13/cobra"
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
	tbbCmd.AddCommand(txCmd())

	if err := tbbCmd.Execute(); err != nil {
		fatal(err)
	}
}
