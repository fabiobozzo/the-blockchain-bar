package main

import (
	"fmt"
	"os"
	"the-blockchain-bar/wallet"

	"github.com/ethereum/go-ethereum/console/prompt"

	"github.com/ethereum/go-ethereum/cmd/utils"

	"github.com/spf13/cobra"
)

func walletCmd() *cobra.Command {
	var walletCmd = &cobra.Command{
		Use:   "wallet",
		Short: "Manages accounts, keys, cryptography.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ErrIncorrectUsage
		},
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	walletCmd.AddCommand(walletNewAccountCmd())

	return walletCmd
}

func walletNewAccountCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "new-account",
		Short: "Creates a new account with a new set of a elliptic-curve Private + Public keys.",
		Run: func(cmd *cobra.Command, args []string) {
			password := getPassPhrase("Please enter a password to encrypt the new wallet:", true)

			dataDir := getDataDirFromCmd(cmd)

			acc, err := wallet.NewKeystoreAccount(dataDir, password)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Printf("New account created: %s\n", acc.Hex())
		},
	}

	addDefaultRequiredFlags(cmd)

	return cmd
}

func getPassPhrase(inputPrompt string, confirmation bool) string {
	fmt.Println(inputPrompt)

	password, err := prompt.Stdin.PromptPassword("Password: ")
	if err != nil {
		utils.Fatalf("Failed to read password: %v", err)
	}

	if confirmation {
		confirm, err := prompt.Stdin.PromptPassword("Repeat password: ")
		if err != nil {
			utils.Fatalf("Failed to read password confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passwords do not match")
		}
	}

	return password
}
