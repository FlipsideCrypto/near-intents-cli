package main

import (
	"time"

	"github.com/spf13/cobra"
)

var (
	flagFrom      string
	flagTo        string
	flagFromChain string
	flagToChain   string
	flagAmount    string
	flagSwapType  string
	flagSlippage  int
	flagRecipient string
	flagRefundTo  string
	flagAppFee    int
	flagFeeRecip  string
	flagNative    bool
	flagSender    string
)

var quoteCmd = &cobra.Command{
	Use:   "quote",
	Short: "Get a swap quote (dry run, no deposit address)",
	Long: `Get a swap quote without generating a deposit address.
Use this to check rates before committing to a swap.

Examples:
  near-intents quote --from USDC --from-chain ethereum --to wNEAR --to-chain near --amount 10
  near-intents quote --from nep141:wrap.near --to nep141:eth-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.omft.near --amount 5`,
	RunE: runQuote,
}

func init() {
	addQuoteFlags(quoteCmd)
	rootCmd.AddCommand(quoteCmd)
}

func addQuoteFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flagFrom, "from", "", "Origin token (asset ID or symbol)")
	cmd.Flags().StringVar(&flagTo, "to", "", "Destination token (asset ID or symbol)")
	cmd.Flags().StringVar(&flagFromChain, "from-chain", "", "Origin blockchain (required when --from is a symbol)")
	cmd.Flags().StringVar(&flagToChain, "to-chain", "", "Destination blockchain (required when --to is a symbol)")
	cmd.Flags().StringVar(&flagAmount, "amount", "", "Human-readable amount (e.g., 1.5)")
	cmd.Flags().StringVar(&flagSwapType, "swap-type", "EXACT_INPUT", "Swap type: EXACT_INPUT, EXACT_OUTPUT, or FLEX_INPUT")
	cmd.Flags().IntVar(&flagSlippage, "slippage", 100, "Slippage tolerance in basis points (100 = 1%)")
	cmd.Flags().StringVar(&flagRecipient, "recipient", "", "Destination address for swapped tokens")
	cmd.Flags().StringVar(&flagRefundTo, "refund-to", "", "Refund address on origin chain")
	cmd.Flags().IntVar(&flagAppFee, "app-fee", 0, "Partner fee in basis points")
	cmd.Flags().StringVar(&flagFeeRecip, "fee-recipient", "", "NEAR address to receive partner fees")
	cmd.Flags().BoolVar(&flagNative, "native", false, "Use NEAR-native intents mode (swap wrapped assets on NEAR)")
	cmd.MarkFlagRequired("from")
	cmd.MarkFlagRequired("to")
	cmd.MarkFlagRequired("amount")
}

func buildQuoteRequest(client *Client, dry bool, deadline time.Duration) (*QuoteRequest, []TokenResponse, error) {
	// Fetch tokens for resolution
	verbose("fetching token list for resolution")
	tokens, err := client.GetTokens()
	if err != nil {
		return nil, nil, &ResolveError{Code: "API_ERROR", Message: "Failed to fetch tokens: " + err.Error()}
	}

	// Default chains to "near" in native mode
	fromChain := flagFromChain
	toChain := flagToChain
	if flagNative {
		if fromChain == "" {
			fromChain = "near"
		}
		if toChain == "" {
			toChain = "near"
		}
	}

	// Resolve from/to tokens
	fromToken, err := resolveTokenByIdOrSymbol(flagFrom, fromChain, tokens)
	if err != nil {
		return nil, nil, err
	}
	toToken, err := resolveTokenByIdOrSymbol(flagTo, toChain, tokens)
	if err != nil {
		return nil, nil, err
	}

	// Convert amount using appropriate token decimals
	decimals := fromToken.Decimals
	if flagSwapType == "EXACT_OUTPUT" {
		decimals = toToken.Decimals
	}
	amountRaw, err := convertAmount(flagAmount, decimals)
	if err != nil {
		return nil, nil, &ResolveError{Code: "INVALID_FLAGS", Message: "Invalid amount: " + err.Error()}
	}

	// Recipient/refund: use provided or placeholder for dry quotes
	recipient := flagRecipient
	refundTo := flagRefundTo
	if dry {
		if recipient == "" {
			recipient = placeholderAddress(toToken.Blockchain)
		}
		if refundTo == "" {
			refundTo = placeholderAddress(fromToken.Blockchain)
		}
	}

	recipientType := "DESTINATION_CHAIN"
	refundType := "ORIGIN_CHAIN"
	depositType := "ORIGIN_CHAIN"
	if flagNative {
		recipientType = "INTENTS"
		refundType = "INTENTS"
		depositType = "INTENTS"
	}

	req := &QuoteRequest{
		Dry:               dry,
		SwapType:          flagSwapType,
		SlippageTolerance: flagSlippage,
		OriginAsset:       fromToken.AssetId,
		DestinationAsset:  toToken.AssetId,
		Amount:            amountRaw,
		Recipient:         recipient,
		RecipientType:     recipientType,
		RefundTo:          refundTo,
		RefundType:        refundType,
		DepositType:       depositType,
		Deadline:          time.Now().Add(deadline),
	}

	// App fees
	if flagAppFee > 0 && flagFeeRecip != "" {
		req.AppFees = []AppFee{
			{Recipient: flagFeeRecip, Fee: float32(flagAppFee)},
		}
	}

	return req, tokens, nil
}

func runQuote(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	client := newClient(cfg)

	req, _, err := buildQuoteRequest(client, true, 1*time.Hour)
	if err != nil {
		if re, ok := err.(*ResolveError); ok {
			PrintErrorResponse(re.Code, re.Message)
			return nil
		}
		PrintErrorResponse("API_ERROR", err.Error())
		return nil
	}

	verbose("requesting dry quote")
	resp, err := client.PostQuote(req)
	if err != nil {
		PrintErrorResponse("QUOTE_FAILED", "Quote request failed: "+err.Error())
		return nil
	}

	PrintSuccess(map[string]interface{}{
		"correlationId":      resp.CorrelationId,
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
	})
	return nil
}
