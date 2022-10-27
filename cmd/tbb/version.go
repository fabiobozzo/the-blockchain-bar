package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Major = "2"
const Minor = "0"
const Fix = "0"
const Verbal = "Tx GAS"

// GitCommit has to be configured via -ldflags during build
var GitCommit string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Describes CLI version.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version is %s.%s.%s-beta %s\n", Major, Minor, Fix, Verbal)
	},
}

func shortGitCommit(fullGitCommit string) string {
	shortCommit := ""
	if len(fullGitCommit) >= 6 {
		shortCommit = fullGitCommit[0:6]
	}

	return shortCommit
}
