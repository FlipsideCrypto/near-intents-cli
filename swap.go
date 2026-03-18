package main

import (
	"time"

	"github.com/spf13/cobra"
)

var flagDeadline time.Duration

var swapCmd = &cobra.Command{
	Use:   "swap",
	Short: "Execute a swap (generates deposit address)",
	Long: `Execute a cross-chain swap. Generates a real deposit address (~10 min validity)
and a signing URL for the user to sign the deposit transaction.

Examples:
  near-intents swap --from USDC --from-chain ethereum --to wNEAR --to-chain near --amount 10 --recipient alice.near --refund-to 0xYourAddr`,
	RunE: runSwap,
}

func init() {
	addQuoteFlags(swapCmd)
	swapCmd.Flags().DurationVar(&flagDeadline, "deadline", 1*time.Hour, "Deadline from now (e.g., 1h, 30m)")
	rootCmd.AddCommand(swapCmd)
}

func runSwap(cmd *cobra.Command, args []string) error {
	if flagRecipient == "" {
		PrintErrorResponse("INVALID_FLAGS", "--recipient is required for swap")
		return nil
	}
	if flagRefundTo == "" {
		PrintErrorResponse("INVALID_FLAGS", "--refund-to is required for swap")
		return nil
	}

	cfg := loadConfig()
	client := newClient(cfg)

	req, tokens, err := buildQuoteRequest(client, false, flagDeadline)
	if err != nil {
		if re, ok := err.(*ResolveError); ok {
			PrintErrorResponse(re.Code, re.Message)
			return nil
		}
		PrintErrorResponse("API_ERROR", err.Error())
		return nil
	}

	verbose("requesting swap quote (dry=false)")
	resp, err := client.PostQuote(req)
	if err != nil {
		PrintErrorResponse("SWAP_FAILED", "Swap request failed: "+err.Error())
		return nil
	}

	// Resolve from token for signing URL
	fromToken, _ := resolveTokenByIdOrSymbol(flagFrom, flagFromChain, tokens)
	contractAddr := ""
	if fromToken.ContractAddress != nil {
		contractAddr = *fromToken.ContractAddress
	}

	signingURL := ""
	if resp.Quote.DepositAddress != nil {
		signingURL = buildSigningURL(SigningParams{
			BaseURL:        cfg.SigningBaseURL,
			Chain:          fromToken.Blockchain,
			DepositAddress: *resp.Quote.DepositAddress,
			Amount:         flagAmount,
			Token:          fromToken.Symbol,
			Decimals:       fromToken.Decimals,
			TokenAddress:   contractAddr,
			AmountUsd:      resp.Quote.AmountInUsd,
		})
	}

	result := map[string]interface{}{
		"correlationId":      resp.CorrelationId,
		"depositAddress":     resp.Quote.DepositAddress,
		"amountIn":           resp.Quote.AmountIn,
		"amountInFormatted":  resp.Quote.AmountInFormatted,
		"amountInUsd":        resp.Quote.AmountInUsd,
		"amountOut":          resp.Quote.AmountOut,
		"amountOutFormatted": resp.Quote.AmountOutFormatted,
		"amountOutUsd":       resp.Quote.AmountOutUsd,
		"minAmountOut":       resp.Quote.MinAmountOut,
		"timeEstimate":       resp.Quote.TimeEstimate,
		"originAsset":        req.OriginAsset,
		"destinationAsset":   req.DestinationAsset,
		"signingUrl":         signingURL,
	}
	if resp.Quote.Deadline != nil {
		result["deadline"] = *resp.Quote.Deadline
	}
	if resp.Quote.DepositMemo != nil {
		result["depositMemo"] = *resp.Quote.DepositMemo
	}

	PrintSuccess(result)
	return nil
}
