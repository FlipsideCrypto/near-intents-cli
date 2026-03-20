package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	mempoolAddressURL = "https://mempool.space/api/address/%s"
	mempoolPricesURL  = "https://mempool.space/api/v1/prices"
)

type mempoolAddressStats struct {
	FundedTxoSum int64 `json:"funded_txo_sum"`
	SpentTxoSum  int64 `json:"spent_txo_sum"`
}

type mempoolAddress struct {
	ChainStats   mempoolAddressStats `json:"chain_stats"`
	MempoolStats mempoolAddressStats `json:"mempool_stats"`
}

type mempoolPrices struct {
	USD float64 `json:"USD"`
}

var mempoolHTTPClient = &http.Client{Timeout: 15 * time.Second}

func mempoolGet(url string, out interface{}) error {
	resp, err := mempoolHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response from %s: %w", url, err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200 from %s: %d", url, resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

// QueryBitcoinBalance fetches the BTC balance for an address via mempool.space.
func QueryBitcoinBalance(address string) (*ChainBalance, error) {
	var addrData mempoolAddress
	if err := mempoolGet(fmt.Sprintf(mempoolAddressURL, address), &addrData); err != nil {
		return nil, fmt.Errorf("mempool address lookup: %w", err)
	}

	confirmedSats := addrData.ChainStats.FundedTxoSum - addrData.ChainStats.SpentTxoSum
	unconfirmedSats := addrData.MempoolStats.FundedTxoSum - addrData.MempoolStats.SpentTxoSum
	totalSats := confirmedSats + unconfirmedSats

	if totalSats == 0 {
		return &ChainBalance{
			Chain:   "bitcoin",
			Address: address,
		}, nil
	}

	btc := float64(totalSats) / 1e8
	balanceStr := trimTrailingZeros(fmt.Sprintf("%.8f", btc))

	var usd float64
	var prices mempoolPrices
	if err := mempoolGet(mempoolPricesURL, &prices); err == nil {
		usd = btc * prices.USD
	}
	// If price fetch fails, usd stays 0 — graceful degradation.

	tb := TokenBalance{
		Symbol:  "BTC",
		Balance: balanceStr,
		Usd:     usd,
	}

	return &ChainBalance{
		Chain:    "bitcoin",
		Address:  address,
		TotalUsd: usd,
		Tokens:   []TokenBalance{tb},
	}, nil
}
