package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Major = "1"
const Minor = "0"
const Fix = "0"
const Verbal = "Secure blockchain with decentralized auth and replay attack prevention."

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Describes CLI version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version is %s.%s.%s-beta %s\n", Major, Minor, Fix, Verbal)
	},
}
