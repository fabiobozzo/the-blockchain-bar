package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Major = "0"
const Minor = "2"
const Fix = "3"
const Verbal = "Immutable Snapshots"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Describes CLI version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version is %s.%s.%s-beta %s\n", Major, Minor, Fix, Verbal)
	},
}
