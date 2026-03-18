package main

import (
	"fmt"
	"net/url"
)

var evmChainIDs = map[string]int{
	"ethereum":  1,
	"base":      8453,
	"arbitrum":  42161,
	"polygon":   137,
	"avalanche": 43114,
	"optimism":  10,
	"bsc":       56,
}

func mapChainToSigningChain(blockchain string) string {
	if _, ok := evmChainIDs[blockchain]; ok {
		return "evm"
	}
	switch blockchain {
	case "solana":
		return "solana"
	case "near":
		return "near"
	default:
		return blockchain
	}
}

type SigningParams struct {
	BaseURL        string
	Chain          string // API blockchain name (e.g., "ethereum", "solana", "near")
	DepositAddress string
	Amount         string // human-readable
	Token          string // symbol
	Decimals       int
	TokenAddress   string // contract address, empty for native
	AmountUsd      string
}

func buildSigningURL(p SigningParams) string {
	params := url.Values{}
	params.Set("chain", mapChainToSigningChain(p.Chain))
	params.Set("deposit", p.DepositAddress)
	params.Set("amount", p.Amount)
	params.Set("token", p.Token)
	params.Set("decimals", fmt.Sprintf("%d", p.Decimals))

	if p.TokenAddress != "" {
		params.Set("tokenAddress", p.TokenAddress)
	}

	if chainID, ok := evmChainIDs[p.Chain]; ok {
		params.Set("chainId", fmt.Sprintf("%d", chainID))
	}

	if p.AmountUsd != "" {
		params.Set("amountUsd", p.AmountUsd)
	}

	return p.BaseURL + "/?" + params.Encode()
}
