package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const HeliusEndpoint = "https://mainnet.helius-rpc.com/"

type heliusRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type heliusAssetsByOwnerParams struct {
	OwnerAddress   string                    `json:"ownerAddress"`
	DisplayOptions heliusDisplayOptions      `json:"displayOptions"`
}

type heliusDisplayOptions struct {
	ShowFungible      bool `json:"showFungible"`
	ShowNativeBalance bool `json:"showNativeBalance"`
}

type heliusResponse struct {
	Result *heliusResult `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type heliusResult struct {
	Items []heliusAsset `json:"items"`
}

type heliusAsset struct {
	Interface string           `json:"interface"`
	Content   heliusContent    `json:"content"`
	TokenInfo heliusTokenInfo  `json:"token_info"`
}

type heliusContent struct {
	Metadata heliusMetadata `json:"metadata"`
}

type heliusMetadata struct {
	Symbol string `json:"symbol"`
}

type heliusTokenInfo struct {
	Symbol    string           `json:"symbol"`
	Balance   int64            `json:"balance"`
	Decimals  int              `json:"decimals"`
	PriceInfo heliusPriceInfo  `json:"price_info"`
}

type heliusPriceInfo struct {
	TotalPrice float64 `json:"total_price"`
}

// QuerySolanaBalances queries Solana token balances for an address via Helius DAS API.
func QuerySolanaBalances(address, apiKey string) (*ChainBalance, error) {
	url := fmt.Sprintf("%s?api-key=%s", HeliusEndpoint, apiKey)

	req := heliusRequest{
		JSONRPC: "2.0",
		ID:      "helius-das",
		Method:  "getAssetsByOwner",
		Params: heliusAssetsByOwnerParams{
			OwnerAddress: address,
			DisplayOptions: heliusDisplayOptions{
				ShowFungible:      true,
				ShowNativeBalance: true,
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal helius request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("helius request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read helius response: %w", err)
	}

	var heliusResp heliusResponse
	if err := json.Unmarshal(body, &heliusResp); err != nil {
		return nil, fmt.Errorf("parse helius response: %w", err)
	}
	if heliusResp.Error != nil {
		return nil, fmt.Errorf("helius API error: %s", heliusResp.Error.Message)
	}
	if heliusResp.Result == nil {
		return nil, fmt.Errorf("helius returned empty result")
	}

	cb := &ChainBalance{
		Chain:   "solana",
		Address: address,
	}

	for _, asset := range heliusResp.Result.Items {
		iface := asset.Interface
		if iface != "FungibleToken" && iface != "FungibleAsset" {
			continue
		}

		symbol := asset.TokenInfo.Symbol
		if symbol == "" {
			symbol = asset.Content.Metadata.Symbol
		}
		if symbol == "" {
			continue
		}

		usd := asset.TokenInfo.PriceInfo.TotalPrice
		balanceStr := formatBalanceStr(fmt.Sprintf("%d", asset.TokenInfo.Balance), asset.TokenInfo.Decimals)
		if balanceStr == "0" || balanceStr == "" {
			continue
		}

		tb := TokenBalance{
			Symbol:  symbol,
			Balance: balanceStr,
			Usd:     usd,
		}
		cb.Tokens = append(cb.Tokens, tb)
		cb.TotalUsd += usd
	}

	return cb, nil
}

// formatBalanceStr converts a raw integer balance string to a decimal string using the given decimals.
func formatBalanceStr(raw string, decimals int) string {
	if decimals == 0 {
		return raw
	}
	var n int64
	fmt.Sscanf(raw, "%d", &n)
	divisor := math.Pow10(decimals)
	val := float64(n) / divisor
	if val == 0 {
		return "0"
	}
	s := fmt.Sprintf("%f", val)
	return trimTrailingZeros(s)
}

// trimTrailingZeros removes trailing zeros after the decimal point.
func trimTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}
