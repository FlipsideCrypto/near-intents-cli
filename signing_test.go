package main

import (
	"net/url"
	"testing"
)

func TestBuildSigningURL(t *testing.T) {
	u := buildSigningURL(SigningParams{
		BaseURL:        "https://swap.example.com",
		Chain:          "ethereum",
		DepositAddress: "0xabc123",
		Amount:         "1.5",
		Token:          "USDC",
		Decimals:       6,
		TokenAddress:   "0xa0b8",
		AmountUsd:      "1.50",
	})
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if q.Get("chain") != "evm" {
		t.Errorf("expected chain=evm, got %s", q.Get("chain"))
	}
	if q.Get("deposit") != "0xabc123" {
		t.Errorf("expected deposit=0xabc123, got %s", q.Get("deposit"))
	}
	if q.Get("chainId") != "1" {
		t.Errorf("expected chainId=1, got %s", q.Get("chainId"))
	}
	if q.Get("tokenAddress") != "0xa0b8" {
		t.Errorf("expected tokenAddress, got %s", q.Get("tokenAddress"))
	}
}

func TestBuildSigningURLNative(t *testing.T) {
	u := buildSigningURL(SigningParams{
		BaseURL:        "https://swap.example.com",
		Chain:          "ethereum",
		DepositAddress: "0xabc123",
		Amount:         "1.0",
		Token:          "ETH",
		Decimals:       18,
		TokenAddress:   "", // native
		AmountUsd:      "3000.00",
	})
	parsed, _ := url.Parse(u)
	q := parsed.Query()
	if q.Get("tokenAddress") != "" {
		t.Error("expected no tokenAddress for native token")
	}
}

func TestBuildSigningURLSolana(t *testing.T) {
	u := buildSigningURL(SigningParams{
		BaseURL:        "https://swap.example.com",
		Chain:          "solana",
		DepositAddress: "So111...",
		Amount:         "1.0",
		Token:          "SOL",
		Decimals:       9,
	})
	parsed, _ := url.Parse(u)
	q := parsed.Query()
	if q.Get("chain") != "solana" {
		t.Errorf("expected chain=solana, got %s", q.Get("chain"))
	}
	if q.Has("chainId") {
		t.Error("expected no chainId for solana")
	}
}
