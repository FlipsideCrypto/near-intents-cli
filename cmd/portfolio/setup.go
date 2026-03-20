package main

import (
	"github.com/spf13/cobra"
)

var (
	flagAdd     bool
	flagRemove  bool
	flagList    bool
	flagChain   string
	flagAddress string
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Manage portfolio addresses",
	Long: `Add, remove, or list wallet addresses in your portfolio configuration.

Examples:
  portfolio setup --add --chain evm --address 0xabc...
  portfolio setup --add --chain near --address alice.near
  portfolio setup --list
  portfolio setup --remove --chain solana --address 7xKX...`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().BoolVar(&flagAdd, "add", false, "Add an address")
	setupCmd.Flags().BoolVar(&flagRemove, "remove", false, "Remove an address")
	setupCmd.Flags().BoolVar(&flagList, "list", false, "List configured addresses")
	setupCmd.Flags().StringVar(&flagChain, "chain", "", "Chain type: near, evm, solana, bitcoin")
	setupCmd.Flags().StringVar(&flagAddress, "address", "", "Wallet address")
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()

	if flagList {
		type listOutput struct {
			Addresses          []Address `json:"addresses"`
			NearIntentsAccount string    `json:"nearIntentsAccount,omitempty"`
		}
		printSuccess(listOutput{
			Addresses:          cfg.Addresses,
			NearIntentsAccount: cfg.NearIntentsAccount,
		})
		return nil
	}

	if flagAdd {
		if flagChain == "" || flagAddress == "" {
			printError("MISSING_FLAGS", "--chain and --address are required with --add")
			return nil
		}
		for _, a := range cfg.Addresses {
			if a.Chain == flagChain && a.Address == flagAddress {
				printError("DUPLICATE", "Address already exists in config")
				return nil
			}
		}
		cfg.Addresses = append(cfg.Addresses, Address{Chain: flagChain, Address: flagAddress})
		if err := saveConfig(cfg); err != nil {
			printError("CONFIG_ERROR", "Failed to save config: "+err.Error())
			return nil
		}
		printSuccess(map[string]string{"status": "added", "chain": flagChain, "address": flagAddress})
		return nil
	}

	if flagRemove {
		if flagChain == "" || flagAddress == "" {
			printError("MISSING_FLAGS", "--chain and --address are required with --remove")
			return nil
		}
		found := false
		var remaining []Address
		for _, a := range cfg.Addresses {
			if a.Chain == flagChain && a.Address == flagAddress {
				found = true
				continue
			}
			remaining = append(remaining, a)
		}
		if !found {
			printError("NOT_FOUND", "Address not found in config")
			return nil
		}
		cfg.Addresses = remaining
		if err := saveConfig(cfg); err != nil {
			printError("CONFIG_ERROR", "Failed to save config: "+err.Error())
			return nil
		}
		printSuccess(map[string]string{"status": "removed", "chain": flagChain, "address": flagAddress})
		return nil
	}

	printError("MISSING_FLAGS", "Specify --add, --remove, or --list")
	return nil
}
