package main

import (
	"testing"
)

func TestResolveByAssetId(t *testing.T) {
	tokens := makeTokens()
	token, err := resolveTokenByIdOrSymbol("nep141:wrap.near", "", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if token.Symbol != "wNEAR" {
		t.Errorf("expected wNEAR, got %s", token.Symbol)
	}
}

func TestResolveBySymbolAndChain(t *testing.T) {
	tokens := makeTokens()
	token, err := resolveTokenByIdOrSymbol("USDC", "ethereum", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if token.Blockchain != "ethereum" {
		t.Errorf("expected ethereum, got %s", token.Blockchain)
	}
}

func TestResolveAmbiguous(t *testing.T) {
	tokens := makeTokens()
	_, err := resolveTokenByIdOrSymbol("USDC", "", tokens)
	if err == nil {
		t.Error("expected TOKEN_AMBIGUOUS error")
	}
	if re, ok := err.(*ResolveError); !ok || re.Code != "TOKEN_AMBIGUOUS" {
		t.Errorf("expected TOKEN_AMBIGUOUS, got %v", err)
	}
}

func TestResolveNotFound(t *testing.T) {
	tokens := makeTokens()
	_, err := resolveTokenByIdOrSymbol("DOGE", "ethereum", tokens)
	if err == nil {
		t.Error("expected TOKEN_NOT_FOUND error")
	}
}

func TestConvertAmount(t *testing.T) {
	// 1.5 USDC with 6 decimals = "1500000"
	result, err := convertAmount("1.5", 6)
	if err != nil {
		t.Fatal(err)
	}
	if result != "1500000" {
		t.Errorf("expected 1500000, got %s", result)
	}
}

func TestConvertAmountWholeNumber(t *testing.T) {
	result, err := convertAmount("10", 24)
	if err != nil {
		t.Fatal(err)
	}
	if result != "10000000000000000000000000" {
		t.Errorf("expected 10 * 10^24, got %s", result)
	}
}

func TestConvertAmountSmallDecimals(t *testing.T) {
	result, err := convertAmount("0.001", 6)
	if err != nil {
		t.Fatal(err)
	}
	if result != "1000" {
		t.Errorf("expected 1000, got %s", result)
	}
}
