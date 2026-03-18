package main

import (
	"testing"
)

func ptr(s string) *string { return &s }

func makeTokens() []TokenResponse {
	return []TokenResponse{
		{AssetId: "nep141:wrap.near", Symbol: "wNEAR", Blockchain: "near", Decimals: 24, Price: 3.5},
		{AssetId: "nep141:eth-0xa0b8.omft.near", Symbol: "USDC", Blockchain: "ethereum", Decimals: 6, Price: 1.0, ContractAddress: ptr("0xa0b8")},
		{AssetId: "nep141:sol-EPjF.omft.near", Symbol: "USDC", Blockchain: "solana", Decimals: 6, Price: 1.0, ContractAddress: ptr("EPjF")},
	}
}

func TestFilterByChain(t *testing.T) {
	tokens := filterTokens(makeTokens(), "ethereum", "")
	if len(tokens) != 1 || tokens[0].Symbol != "USDC" {
		t.Errorf("expected 1 USDC on ethereum, got %d tokens", len(tokens))
	}
}

func TestFilterBySearch(t *testing.T) {
	tokens := filterTokens(makeTokens(), "", "usdc")
	if len(tokens) != 2 {
		t.Errorf("expected 2 USDC tokens, got %d", len(tokens))
	}
}

func TestFilterByChainAndSearch(t *testing.T) {
	tokens := filterTokens(makeTokens(), "solana", "usdc")
	if len(tokens) != 1 || tokens[0].Blockchain != "solana" {
		t.Errorf("expected 1 USDC on solana, got %d", len(tokens))
	}
}

func TestFilterNoMatch(t *testing.T) {
	tokens := filterTokens(makeTokens(), "", "DOGE")
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}
