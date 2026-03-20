package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// ---- output types ----

type balancesOutput struct {
	TotalUsd float64        `json:"totalUsd"`
	Chains   []chainBalance `json:"chains"`
}

type chainBalance struct {
	Chain    string         `json:"chain"`
	Address  string         `json:"address"`
	TotalUsd float64        `json:"totalUsd"`
	Tokens   []tokenBalance `json:"tokens"`
}

type tokenBalance struct {
	Symbol          string  `json:"symbol"`
	Balance         string  `json:"balance"`
	Usd             float64 `json:"usd"`
	ContractAddress string  `json:"contractAddress,omitempty"`
	AssetId         string  `json:"assetId,omitempty"`
}

// ---- command ----

var flagAccount string

var balancesCmd = &cobra.Command{
	Use:   "balances",
	Short: "Show NEAR wallet and intents balances",
	Long: `Show token balances for a NEAR account, across both the on-chain wallet
and the NEAR Intents deposit contract.

Examples:
  near-intents balances --account alice.near
  near-intents balances --account alice.near --pretty`,
	RunE: runBalances,
}

func init() {
	balancesCmd.Flags().StringVar(&flagAccount, "account", "", "NEAR account ID (required)")
	_ = balancesCmd.MarkFlagRequired("account")
	rootCmd.AddCommand(balancesCmd)
}

// ---- NEAR RPC helpers ----

const nearRPCEndpoint = "https://rpc.mainnet.near.org"

type nearRPCRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type nearRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func doNearRPC(method string, params any) (json.RawMessage, error) {
	reqBody := nearRPCRequest{
		Jsonrpc: "2.0",
		ID:      "near-intents-cli",
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	verbose("NEAR RPC %s %s", nearRPCEndpoint, method)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Post(nearRPCEndpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("near rpc request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("near rpc read body: %w", err)
	}

	var rpcResp nearRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("near rpc parse response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("near rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

// ---- native NEAR balance ----

func fetchNativeNEAR(accountID string) (string, error) {
	params := map[string]string{
		"request_type": "view_account",
		"finality":     "final",
		"account_id":   accountID,
	}
	result, err := doNearRPC("query", params)
	if err != nil {
		return "", err
	}
	var account struct {
		Amount string `json:"amount"`
	}
	if err := json.Unmarshal(result, &account); err != nil {
		return "", fmt.Errorf("parse view_account: %w", err)
	}
	return account.Amount, nil
}

// ---- FastNEAR FT discovery ----

type fastNEARToken struct {
	ContractID string `json:"contract_id"`
	Balance    string `json:"balance"`
}

func fetchFastNEARTokens(accountID string) ([]fastNEARToken, error) {
	url := fmt.Sprintf("https://api.fastnear.com/v1/account/%s/ft", accountID)
	verbose("FastNEAR GET %s", url)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fastnear request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fastnear read body: %w", err)
	}

	var result struct {
		Tokens []fastNEARToken `json:"tokens"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("fastnear parse: %w", err)
	}
	return result.Tokens, nil
}

// ---- intents balances ----

func fetchIntentsBalances(accountID string, tokenIDs []string) ([]string, error) {
	argsJSON, err := json.Marshal(map[string]any{
		"account_id": accountID,
		"token_ids":  tokenIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal intents args: %w", err)
	}

	params := map[string]any{
		"request_type": "call_function",
		"finality":     "final",
		"account_id":   "intents.near",
		"method_name":  "mt_batch_balance_of",
		"args_base64":  base64.StdEncoding.EncodeToString(argsJSON),
	}

	result, err := doNearRPC("query", params)
	if err != nil {
		return nil, err
	}

	// result is {"result":[...bytes...], "logs":[], ...}
	var callResult struct {
		Result []byte `json:"result"`
	}
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("parse call_function result wrapper: %w", err)
	}

	// callResult.Result is a JSON array of balance strings
	var balances []string
	if err := json.Unmarshal(callResult.Result, &balances); err != nil {
		return nil, fmt.Errorf("parse intents balances: %w", err)
	}
	return balances, nil
}

// ---- balance formatting ----

// formatBalance converts a raw integer string to a human-readable decimal string
// using the given number of decimal places.
func formatBalance(rawAmount string, decimals int) string {
	if rawAmount == "" || rawAmount == "0" {
		return "0"
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(rawAmount, 10); !ok {
		return "0"
	}

	// divisor = 10^decimals, computed with big.Int to avoid float precision loss
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)

	// Use big.Float with high precision
	f := new(big.Float).SetPrec(256)
	f.SetInt(amount)
	d := new(big.Float).SetPrec(256)
	d.SetInt(divisor)
	f.Quo(f, d)

	return f.Text('f', 6)
}

// balanceToFloat converts a raw integer string to float64 for USD calculations.
func balanceToFloat(rawAmount string, decimals int) float64 {
	formatted := formatBalance(rawAmount, decimals)
	f := new(big.Float).SetPrec(256)
	f.SetString(formatted) //nolint:errcheck
	result, _ := f.Float64()
	return result
}

// ---- orchestrator ----

func runBalances(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()
	client := newClient(cfg)

	verbose("fetching token registry")
	tokens, err := client.GetTokens()
	if err != nil {
		PrintErrorResponse("API_ERROR", "Failed to fetch token registry: "+err.Error())
		return nil
	}

	// Build lookup maps keyed by contractAddress (lowercase) for NEAR tokens
	type tokenMeta struct {
		symbol   string
		decimals int
		price    float64
		assetID  string
	}
	metaByContract := make(map[string]tokenMeta)
	var intentTokenIDs []string
	var nearNativePrice float64

	for _, t := range tokens {
		if t.Blockchain != "near" {
			continue
		}
		if t.ContractAddress == nil {
			// This is the native NEAR token entry
			nearNativePrice = float64(t.Price)
			continue
		}
		ca := *t.ContractAddress
		metaByContract[ca] = tokenMeta{
			symbol:   t.Symbol,
			decimals: t.Decimals,
			price:    float64(t.Price),
			assetID:  t.AssetId,
		}
		intentTokenIDs = append(intentTokenIDs, "nep141:"+ca)
	}

	// Fan out 3 queries in parallel
	var (
		mu             sync.Mutex
		nativeAmount   string
		nativeErr      error
		ftTokens       []fastNEARToken
		ftErr          error
		intentBalances []string
		intentErr      error
		wg             sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		amount, e := fetchNativeNEAR(flagAccount)
		mu.Lock()
		nativeAmount = amount
		nativeErr = e
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		toks, e := fetchFastNEARTokens(flagAccount)
		mu.Lock()
		ftTokens = toks
		ftErr = e
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		if len(intentTokenIDs) == 0 {
			return
		}
		balances, e := fetchIntentsBalances(flagAccount, intentTokenIDs)
		mu.Lock()
		intentBalances = balances
		intentErr = e
		mu.Unlock()
	}()

	wg.Wait()

	if nativeErr != nil {
		verbose("native NEAR balance error: %v", nativeErr)
	}
	if ftErr != nil {
		verbose("FastNEAR FT error: %v", ftErr)
	}
	if intentErr != nil {
		verbose("intents balance error: %v", intentErr)
	}

	// ---- Build near (wallet) chain entry ----
	var nearTokens []tokenBalance
	var nearTotalUsd float64

	// Native NEAR (24 decimals)
	if nativeAmount != "" && nativeErr == nil {
		bal := formatBalance(nativeAmount, 24)
		usd := balanceToFloat(nativeAmount, 24) * nearNativePrice
		nearTokens = append(nearTokens, tokenBalance{
			Symbol:  "NEAR",
			Balance: bal,
			Usd:     usd,
			AssetId: "near:native",
		})
		nearTotalUsd += usd
	}

	// FT tokens from FastNEAR
	if ftErr == nil {
		for _, ft := range ftTokens {
			if ft.Balance == "0" || ft.Balance == "" {
				continue
			}
			meta, ok := metaByContract[ft.ContractID]
			if !ok {
				verbose("skipping unknown FT contract: %s", ft.ContractID)
				continue
			}
			bal := formatBalance(ft.Balance, meta.decimals)
			usd := balanceToFloat(ft.Balance, meta.decimals) * meta.price
			nearTokens = append(nearTokens, tokenBalance{
				Symbol:          meta.symbol,
				Balance:         bal,
				Usd:             usd,
				ContractAddress: ft.ContractID,
				AssetId:         meta.assetID,
			})
			nearTotalUsd += usd
		}
	}

	nearChain := chainBalance{
		Chain:    "near",
		Address:  flagAccount,
		TotalUsd: nearTotalUsd,
		Tokens:   nearTokens,
	}

	// ---- Build near-intents chain entry ----
	var intentsTokens []tokenBalance
	var intentsTotalUsd float64

	for i, tokenID := range intentTokenIDs {
		if i >= len(intentBalances) {
			break
		}
		bal := intentBalances[i]
		if bal == "0" || bal == "" {
			continue
		}
		// Strip "nep141:" prefix to get contract address
		contractAddr := tokenID[len("nep141:"):]
		meta, ok := metaByContract[contractAddr]
		if !ok {
			continue
		}
		formatted := formatBalance(bal, meta.decimals)
		usd := balanceToFloat(bal, meta.decimals) * meta.price
		intentsTokens = append(intentsTokens, tokenBalance{
			Symbol:          meta.symbol,
			Balance:         formatted,
			Usd:             usd,
			ContractAddress: contractAddr,
			AssetId:         meta.assetID,
		})
		intentsTotalUsd += usd
	}

	intentsChain := chainBalance{
		Chain:    "near-intents",
		Address:  "intents.near",
		TotalUsd: intentsTotalUsd,
		Tokens:   intentsTokens,
	}

	out := balancesOutput{
		TotalUsd: nearTotalUsd + intentsTotalUsd,
		Chains:   []chainBalance{nearChain, intentsChain},
	}

	PrintSuccess(out)
	return nil
}
