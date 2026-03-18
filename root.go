package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagToken   string
	flagPretty  bool
	flagVerbose bool
)

var rootCmd = &cobra.Command{
	Use:     "near-intents",
	Short:   "CLI for cross-chain token swaps via NEAR Intents",
	Long:    "A CLI tool wrapping the Defuse Protocol 1Click Swap API for cross-chain token swaps.",
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		prettyOutput = flagPretty
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "JWT bearer token for authentication")
	rootCmd.PersistentFlags().BoolVar(&flagPretty, "pretty", false, "Pretty-print JSON output with indentation")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Verbose logging to stderr")
	rootCmd.SetVersionTemplate(fmt.Sprintf("near-intents %s (commit: %s, built: %s)\n", version, commit, date))
}
