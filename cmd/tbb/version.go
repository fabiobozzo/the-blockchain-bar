package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Major = "2"
const Minor = "0"
const Fix = "0"
const Verbal = "Tx GAS"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Describes CLI version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version is %s.%s.%s-beta %s\n", Major, Minor, Fix, Verbal)
	},
}
