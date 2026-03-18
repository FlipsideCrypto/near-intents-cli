package main

import (
	oneclick "github.com/defuse-protocol/one-click-sdk-go"
	"github.com/spf13/cobra"
)

var (
	submitDepositAddr string
	submitTxHash      string
	submitNearSender  string
	submitMemo        string
)

var submitTxCmd = &cobra.Command{
	Use:   "submit-tx",
	Short: "Submit deposit transaction hash (optional, speeds up processing)",
	Long: `Notify the service that a deposit has been made.
This is optional but speeds up swap processing.

Examples:
  near-intents submit-tx --deposit-address 0xabc... --tx-hash 0xdef...
  near-intents submit-tx --deposit-address 0xabc... --tx-hash 0xdef... --near-sender alice.near`,
	RunE: runSubmitTx,
}

func init() {
	submitTxCmd.Flags().StringVar(&submitDepositAddr, "deposit-address", "", "Deposit address from swap output")
	submitTxCmd.Flags().StringVar(&submitTxHash, "tx-hash", "", "Transaction hash of the deposit")
	submitTxCmd.Flags().StringVar(&submitNearSender, "near-sender", "", "NEAR account that sent the deposit (NEAR chain only)")
	submitTxCmd.Flags().StringVar(&submitMemo, "memo", "", "Deposit memo (Stellar chain only)")
	submitTxCmd.MarkFlagRequired("deposit-address")
	submitTxCmd.MarkFlagRequired("tx-hash")
	rootCmd.AddCommand(submitTxCmd)
}

func runSubmitTx(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	client := newClient(cfg)

	req := oneclick.SubmitDepositTxRequest{
		TxHash:         submitTxHash,
		DepositAddress: submitDepositAddr,
	}
	if submitNearSender != "" {
		req.NearSenderAccount = &submitNearSender
	}
	if submitMemo != "" {
		req.Memo = &submitMemo
	}

	verbose("submitting deposit tx %s for %s", submitTxHash, submitDepositAddr)
	resp, _, err := client.api.OneClickAPI.SubmitDepositTx(client.ctx()).SubmitDepositTxRequest(req).Execute()
	if err != nil {
		PrintErrorResponse("API_ERROR", "Failed to submit deposit tx: "+err.Error())
		return nil
	}

	PrintSuccess(map[string]interface{}{
		"correlationId": resp.CorrelationId,
		"status":        resp.Status,
	})
	return nil
}
