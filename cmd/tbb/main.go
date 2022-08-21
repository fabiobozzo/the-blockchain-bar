package main

import (
	"errors"
	"fmt"
	"os"

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

	if err := tbbCmd.Execute(); err != nil {
		fmt.Println(os.Stderr, err)
		os.Exit(1)
	}
}
