package main

import (
	"strings"
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
	swapCmd.Flags().StringVar(&flagSender, "sender", "", "NEAR account that will sign the transaction (required with --native)")
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
	if flagNative {
		if flagSender == "" {
			PrintErrorResponse("INVALID_FLAGS", "--sender is required for native swap")
			return nil
		}
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

	// Validate both tokens are on NEAR in native mode (before making API call)
	if flagNative {
		if !strings.HasPrefix(req.OriginAsset, "nep141:") || !strings.HasPrefix(req.DestinationAsset, "nep141:") {
			PrintErrorResponse("INVALID_FLAGS", "both tokens must be on the near blockchain when using --native")
			return nil
		}
	}

	verbose("requesting swap quote (dry=false)")
	resp, err := client.PostQuote(req)
	if err != nil {
		PrintErrorResponse("SWAP_FAILED", "Swap request failed: "+err.Error())
		return nil
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
	}

	if flagNative {
		// Build nearTransaction for ft_transfer_call
		contractId := strings.TrimPrefix(req.OriginAsset, "nep141:")
		depositAddr := ""
		if resp.Quote.DepositAddress != nil {
			depositAddr = *resp.Quote.DepositAddress
		}
		result["nearTransaction"] = map[string]interface{}{
			"contractId": contractId,
			"method":     "ft_transfer_call",
			"args": map[string]interface{}{
				"receiver_id": "intents.near",
				"amount":      resp.Quote.AmountIn,
				"msg":         depositAddr,
			},
			"gas":      "100 Tgas",
			"deposit":  "1 yoctoNEAR",
			"signerId": flagSender,
		}
	} else {
		// Build signing URL for cross-chain swap
		fromToken, _ := resolveTokenByIdOrSymbol(flagFrom, flagFromChain, tokens)
		contractAddr := ""
		if fromToken.ContractAddress != nil {
			contractAddr = *fromToken.ContractAddress
		}
		if resp.Quote.DepositAddress != nil {
			result["signingUrl"] = buildSigningURL(SigningParams{
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
