package main

import (
	"fmt"
	"the-blockchain-bar/database"
	"the-blockchain-bar/node"

	"github.com/spf13/cobra"
)

func balancesCmd() *cobra.Command {
	var balancesCmd = &cobra.Command{
		Use:   "balances",
		Short: "Interact with balances (list, ...).",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ErrIncorrectUsage
		},
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	balancesCmd.AddCommand(balancesListCmd())

	return balancesCmd
}

func balancesListCmd() *cobra.Command {
	balancesListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all balances.",
		Run: func(cmd *cobra.Command, args []string) {
			state, err := database.NewStateFromDisk(getDataDirFromCmd(cmd), node.DefaultMiningDifficulty)
			if err != nil {
				fatal(err)
			}
			defer state.Close()

			fmt.Printf("Accounts balances at %x:\n", state.LatestBlockHash())
			fmt.Println("-----------------")
			fmt.Println("")

			for account, balance := range state.Balances {
				fmt.Println(fmt.Sprintf("%s: %d", account.String(), balance))
			}

			fmt.Println("")
			fmt.Printf("Accounts nonces:")
			fmt.Println("")
			fmt.Println("__________________")
			fmt.Println("")
			for account, nonce := range state.AccountToNonce {
				fmt.Println(fmt.Sprintf("%s: %d", account.String(), nonce))
			}
		},
	}

	addDefaultRequiredFlags(balancesListCmd)

	return balancesListCmd
}
