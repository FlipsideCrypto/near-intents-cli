package main

import (
	"github.com/FlipsideCrypto/near-intents-cli/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update portfolio to the latest release",
	RunE: func(cmd *cobra.Command, args []string) error {
		return updater.RunUpdate("portfolio", version)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
