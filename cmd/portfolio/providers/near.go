package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

const (
	NearRPCEndpoint = "https://rpc.mainnet.near.org"
	FastNEARBaseURL = "https://api.fastnear.com"
)

// NEARTokenInfo holds metadata and pricing for a NEAR fungible token.
type NEARTokenInfo struct {
	Symbol   string
	Decimals int
	Price    float32
	AssetId  string
}

type nearRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type nearRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error"`
}

// FastNEARToken represents a fungible token entry from the FastNEAR API.
type FastNEARToken struct {
	ContractID string `json:"contract_id"`
	Balance    string `json:"balance"`
}

type fastNEARFTResponse struct {
	Tokens []FastNEARToken `json:"tokens"`
}

var nearHTTPClient = &http.Client{Timeout: 20 * time.Second}

// doNearRPC performs a NEAR JSON-RPC call and returns the raw result bytes.
func doNearRPC(method string, params interface{}) (json.RawMessage, error) {
	req := nearRPCRequest{
		JSONRPC: "2.0",
		ID:      "near-cli",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal near rpc request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", NearRPCEndpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := nearHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("near rpc request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read near rpc response: %w", err)
	}

	var rpcResp nearRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse near rpc response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("near rpc error: %s (data: %s)", rpcResp.Error.Message, rpcResp.Error.Data)
	}

	return rpcResp.Result, nil
}

// viewAccountResult is the response shape for view_account RPC.
type viewAccountResult struct {
	Amount string `json:"amount"`
}

// QueryNEARBalances fetches NEAR native + FT balances for an account.
func QueryNEARBalances(account string, tokenRegistry map[string]NEARTokenInfo, nearPrice float32) (*ChainBalance, error) {
	cb := &ChainBalance{
		Chain:   "near",
		Address: account,
	}

	// 1. Native NEAR balance.
	nativeResult, err := doNearRPC("query", map[string]interface{}{
		"request_type": "view_account",
		"finality":     "final",
		"account_id":   account,
	})
	if err != nil {
		return nil, fmt.Errorf("view_account rpc: %w", err)
	}

	var acct viewAccountResult
	if err := json.Unmarshal(nativeResult, &acct); err != nil {
		return nil, fmt.Errorf("parse view_account result: %w", err)
	}

	nearDecimals := 24
	nearBalanceStr := FormatBalance(acct.Amount, nearDecimals)
	nearBalF := ParseFloat(nearBalanceStr)
	nearUSD := nearBalF * float64(nearPrice)

	if nearBalF > 0 {
		tb := TokenBalance{
			Symbol:  "NEAR",
			Balance: nearBalanceStr,
			Usd:     nearUSD,
		}
		cb.Tokens = append(cb.Tokens, tb)
		cb.TotalUsd += nearUSD
	}

	// 2. Fungible token balances via FastNEAR.
	ftURL := fmt.Sprintf("%s/v1/account/%s/ft", FastNEARBaseURL, account)
	ftResp, err := nearHTTPClient.Get(ftURL)
	if err != nil {
		// Graceful degradation: return native balance only.
		return cb, nil
	}
	defer ftResp.Body.Close()

	ftBody, err := io.ReadAll(ftResp.Body)
	if err != nil || ftResp.StatusCode != 200 {
		return cb, nil
	}

	var ftData fastNEARFTResponse
	if err := json.Unmarshal(ftBody, &ftData); err != nil {
		return cb, nil
	}

	for _, ft := range ftData.Tokens {
		info, ok := tokenRegistry[ft.ContractID]
		if !ok {
			continue
		}
		if ft.Balance == "" || ft.Balance == "0" {
			continue
		}

		balStr := FormatBalance(ft.Balance, info.Decimals)
		balF := ParseFloat(balStr)
		if balF == 0 {
			continue
		}

		usd := balF * float64(info.Price)

		tb := TokenBalance{
			Symbol:          info.Symbol,
			Balance:         balStr,
			Usd:             usd,
			ContractAddress: ft.ContractID,
			AssetId:         info.AssetId,
		}
		cb.Tokens = append(cb.Tokens, tb)
		cb.TotalUsd += usd
	}

	return cb, nil
}

// FormatBalance converts a raw integer balance string to a human-readable decimal string.
// Uses big.Int arithmetic to avoid float64 precision loss for large decimals (e.g. NEAR's 24).
func FormatBalance(raw string, decimals int) string {
	if decimals == 0 {
		return raw
	}
	bal := new(big.Float).SetPrec(256)
	bal.SetString(raw)
	divisor := new(big.Float).SetPrec(256).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil),
	)
	bal.Quo(bal, divisor)
	return bal.Text('f', -1)
}

// ParseFloat parses a decimal string to float64.
func ParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
