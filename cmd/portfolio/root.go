package main

import (
	"fmt"
	"os"

	"github.com/FlipsideCrypto/near-intents-cli/internal/output"
	"github.com/FlipsideCrypto/near-intents-cli/internal/updater"
	"github.com/spf13/cobra"
)

var (
	flagPretty  bool
	flagVerbose bool
	exitCode    int
)

var rootCmd = &cobra.Command{
	Use:     "portfolio",
	Short:   "Multi-chain portfolio balance viewer",
	Long:    "A CLI tool for querying token balances across NEAR, EVM chains, Solana, and Bitcoin.",
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.PrettyOutput = flagPretty
		if cmd.Name() != "update" && cmd.Name() != "version" {
			updater.AutoCheck("portfolio", version)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagPretty, "pretty", false, "Pretty-print JSON output")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Verbose logging to stderr")
	rootCmd.SetVersionTemplate(fmt.Sprintf("portfolio %s (commit: %s, built: %s)\n", version, commit, date))
}

func printSuccess(data interface{}) {
	output.PrintSuccess(data)
}

func printError(code, message string) {
	output.PrintErrorResponse(code, message)
	exitCode = output.ExitCode
}

func verbose(format string, args ...any) {
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}
