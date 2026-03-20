package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const AnkrEndpoint = "https://rpc.ankr.com/multichain"

type ankrRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type ankrBalanceParams struct {
	Wallets []string `json:"walletAddress"`
}

type ankrResponse struct {
	Result *ankrResult `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type ankrResult struct {
	TotalBalanceUsd string      `json:"totalBalanceUsd"`
	Assets          []ankrAsset `json:"assets"`
}

type ankrAsset struct {
	Blockchain      string `json:"blockchain"`
	TokenName       string `json:"tokenName"`
	TokenSymbol     string `json:"tokenSymbol"`
	TokenDecimals   int    `json:"tokenDecimals"`
	TokenType       string `json:"tokenType"`
	ContractAddress string `json:"contractAddress,omitempty"`
	Balance         string `json:"balance"`
	BalanceRawInt   string `json:"balanceRawInteger"`
	BalanceUsd      string `json:"balanceUsd"`
	TokenPrice      string `json:"tokenPrice"`
}

// QueryEVMBalances queries all EVM chain balances for an address via Ankr.
func QueryEVMBalances(address, apiKey string) ([]ChainBalance, error) {
	url := fmt.Sprintf("%s/%s", AnkrEndpoint, apiKey)

	req := ankrRequest{
		JSONRPC: "2.0",
		Method:  "ankr_getAccountBalance",
		Params:  ankrBalanceParams{Wallets: []string{address}},
		ID:      1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal ankr request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ankr request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ankr response: %w", err)
	}

	var ankrResp ankrResponse
	if err := json.Unmarshal(body, &ankrResp); err != nil {
		return nil, fmt.Errorf("parse ankr response: %w", err)
	}
	if ankrResp.Error != nil {
		return nil, fmt.Errorf("ankr API error: %s", ankrResp.Error.Message)
	}
	if ankrResp.Result == nil {
		return nil, fmt.Errorf("ankr returned empty result")
	}

	// Group assets by blockchain
	chainMap := make(map[string]*ChainBalance)
	for _, asset := range ankrResp.Result.Assets {
		cb, ok := chainMap[asset.Blockchain]
		if !ok {
			cb = &ChainBalance{
				Chain:   asset.Blockchain,
				Address: address,
			}
			chainMap[asset.Blockchain] = cb
		}

		usd := parseFloat(asset.BalanceUsd)
		tb := TokenBalance{
			Symbol:  asset.TokenSymbol,
			Balance: asset.Balance,
			Usd:     usd,
		}
		if asset.ContractAddress != "" && asset.TokenType != "NATIVE" {
			tb.ContractAddress = asset.ContractAddress
		}
		cb.Tokens = append(cb.Tokens, tb)
		cb.TotalUsd += usd
	}

	var result []ChainBalance
	for _, cb := range chainMap {
		if len(cb.Tokens) > 0 {
			result = append(result, *cb)
		}
	}
	return result, nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
