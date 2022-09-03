package main

import (
	"fmt"
	"the-blockchain-bar/database"

	"github.com/spf13/cobra"
)

const (
	flagFrom  = "from"
	flagTo    = "to"
	flagValue = "value"
	flagData  = "data"
)

func txCmd() *cobra.Command {
	var txsCmd = &cobra.Command{
		Use:   "tx",
		Short: "Interact with transactions (add, ...).",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ErrIncorrectUsage
		},
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	txsCmd.AddCommand(txAddCmd())

	return txsCmd
}

func txAddCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "add",
		Short: "Adds new TX to database.",
		Run: func(cmd *cobra.Command, args []string) {
			from, _ := cmd.Flags().GetString(flagFrom)
			to, _ := cmd.Flags().GetString(flagTo)
			value, _ := cmd.Flags().GetUint(flagValue)
			data, _ := cmd.Flags().GetString(flagData)

			tx := database.NewTx(
				database.NewAccount(from),
				database.NewAccount(to),
				value,
				data,
			)

			state, err := database.NewStateFromDisk()
			if err != nil {
				fatal(err)
			}

			defer state.Close()

			if err := state.AddTx(tx); err != nil {
				fatal(err)
			}

			if _, err := state.Persist(); err != nil {
				fatal(err)
			}

			fmt.Println("TX successfully persisted to the ledger.")
		},
	}

	cmd.Flags().String(flagFrom, "", "From what account to send tokens.")
	cmd.MarkFlagRequired(flagFrom)

	cmd.Flags().String(flagTo, "", "To what account to send tokens.")
	cmd.MarkFlagRequired(flagTo)

	cmd.Flags().Uint(flagValue, 0, "How many tokens to send.")
	cmd.MarkFlagRequired(flagValue)

	cmd.Flags().String(flagData, "", "Possible values: 'reward'.")

	return cmd
}
