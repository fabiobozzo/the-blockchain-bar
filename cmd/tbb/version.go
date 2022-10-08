package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Major = "0"
const Minor = "10"
const Fix = "0"
const Verbal = "Accounts as crypto public keys"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Describes CLI version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version is %s.%s.%s-beta %s\n", Major, Minor, Fix, Verbal)
	},
}
