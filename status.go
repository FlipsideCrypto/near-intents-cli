package main

import (
	"github.com/spf13/cobra"
)

var (
	statusDepositAddr string
	statusDepositMemo string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check swap progress",
	Long: `Check the current status of a swap by deposit address.
Returns a single status check (no polling).

Terminal states: SUCCESS, FAILED, REFUNDED, INCOMPLETE_DEPOSIT
Non-terminal states: PENDING_DEPOSIT, KNOWN_DEPOSIT_TX, PROCESSING

Examples:
  near-intents status --deposit-address 0xabc...`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusDepositAddr, "deposit-address", "", "Deposit address to check")
	statusCmd.Flags().StringVar(&statusDepositMemo, "deposit-memo", "", "Deposit memo (required for Stellar)")
	statusCmd.MarkFlagRequired("deposit-address")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	client := newClient(cfg)

	verbose("checking status for %s", statusDepositAddr)
	apiCall := client.api.OneClickAPI.GetExecutionStatus(client.ctx()).DepositAddress(statusDepositAddr)
	if statusDepositMemo != "" {
		apiCall = apiCall.DepositMemo(statusDepositMemo)
	}

	resp, _, err := apiCall.Execute()
	if err != nil {
		PrintErrorResponse("API_ERROR", "Failed to get status: "+err.Error())
		return nil
	}

	result := map[string]interface{}{
		"correlationId": resp.CorrelationId,
		"status":        resp.Status,
		"updatedAt":     resp.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Include swap details when available
	sd := resp.SwapDetails
	details := map[string]interface{}{}
	if sd.AmountIn != nil {
		details["amountIn"] = *sd.AmountIn
	}
	if sd.AmountInFormatted != nil {
		details["amountInFormatted"] = *sd.AmountInFormatted
	}
	if sd.AmountInUsd != nil {
		details["amountInUsd"] = *sd.AmountInUsd
	}
	if sd.AmountOut != nil {
		details["amountOut"] = *sd.AmountOut
	}
	if sd.AmountOutFormatted != nil {
		details["amountOutFormatted"] = *sd.AmountOutFormatted
	}
	if sd.AmountOutUsd != nil {
		details["amountOutUsd"] = *sd.AmountOutUsd
	}
	if len(sd.OriginChainTxHashes) > 0 {
		details["originChainTxHashes"] = sd.OriginChainTxHashes
	}
	if len(sd.DestinationChainTxHashes) > 0 {
		details["destinationChainTxHashes"] = sd.DestinationChainTxHashes
	}
	if sd.RefundedAmount != nil {
		details["refundedAmount"] = *sd.RefundedAmount
		if sd.RefundReason != nil {
			details["refundReason"] = *sd.RefundReason
		}
	}
	if len(details) > 0 {
		result["swapDetails"] = details
	}

	PrintSuccess(result)
	return nil
}
