package main

import (
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagChain  string
	flagSearch string
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "List supported swap tokens",
	Long: `List all tokens supported by the NEAR Intents swap API.
Filtering is client-side — all tokens are fetched, then filtered locally.

Examples:
  near-intents tokens
  near-intents tokens --chain ethereum
  near-intents tokens --search USDC
  near-intents tokens --chain near --search wNEAR`,
	RunE: runTokens,
}

func init() {
	tokensCmd.Flags().StringVar(&flagChain, "chain", "", "Filter by blockchain (e.g., ethereum, solana, near)")
	tokensCmd.Flags().StringVar(&flagSearch, "search", "", "Fuzzy match on symbol, blockchain, or asset ID")
	rootCmd.AddCommand(tokensCmd)
}

func runTokens(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	client := newClient(cfg)

	verbose("fetching tokens from %s", cfg.APIEndpoint)
	tokens, err := client.GetTokens()
	if err != nil {
		PrintErrorResponse("API_ERROR", "Failed to fetch tokens: "+err.Error())
		return nil
	}

	filtered := filterTokens(tokens, flagChain, flagSearch)
	verbose("found %d tokens (filtered from %d)", len(filtered), len(tokens))

	type tokenOutput struct {
		AssetId         string  `json:"assetId"`
		Symbol          string  `json:"symbol"`
		Blockchain      string  `json:"blockchain"`
		Decimals        int     `json:"decimals"`
		Price           float32 `json:"price"`
		PriceUpdatedAt  string  `json:"priceUpdatedAt,omitempty"`
		ContractAddress *string `json:"contractAddress,omitempty"`
	}

	out := make([]tokenOutput, len(filtered))
	for i, t := range filtered {
		out[i] = tokenOutput{
			AssetId:         t.AssetId,
			Symbol:          t.Symbol,
			Blockchain:      t.Blockchain,
			Decimals:        int(t.Decimals),
			Price:           t.Price,
			PriceUpdatedAt:  t.PriceUpdatedAt,
			ContractAddress: t.ContractAddress,
		}
	}
	PrintSuccess(out)
	return nil
}

func filterTokens(tokens []TokenResponse, chain, search string) []TokenResponse {
	if chain == "" && search == "" {
		return tokens
	}

	chain = strings.ToLower(chain)
	search = strings.ToLower(search)

	var result []TokenResponse
	for _, t := range tokens {
		if chain != "" && strings.ToLower(t.Blockchain) != chain {
			continue
		}
		if search != "" {
			match := strings.Contains(strings.ToLower(t.Symbol), search) ||
				strings.Contains(strings.ToLower(t.Blockchain), search) ||
				strings.Contains(strings.ToLower(t.AssetId), search)
			if !match {
				continue
			}
		}
		result = append(result, t)
	}
	return result
}
