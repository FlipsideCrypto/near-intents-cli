package main

import (
	"fmt"
	"math/big"
	"strings"
)

type ResolveError struct {
	Code    string
	Message string
}

func (e *ResolveError) Error() string {
	return e.Message
}

// resolveTokenByIdOrSymbol resolves a token identifier to a TokenResponse.
// If identifier looks like an asset ID (contains ":"), match exactly.
// Otherwise treat as symbol and require chain for disambiguation.
func resolveTokenByIdOrSymbol(identifier, chain string, tokens []TokenResponse) (*TokenResponse, error) {
	// Asset ID mode: exact match
	if strings.Contains(identifier, ":") {
		for i, t := range tokens {
			if t.AssetId == identifier {
				return &tokens[i], nil
			}
		}
		return nil, &ResolveError{Code: "TOKEN_NOT_FOUND", Message: fmt.Sprintf("no token with asset ID %q", identifier)}
	}

	// Symbol mode: match symbol, optionally filter by chain
	symbol := strings.ToLower(identifier)
	var matches []int
	for i, t := range tokens {
		if strings.ToLower(t.Symbol) != symbol {
			continue
		}
		if chain != "" && strings.ToLower(t.Blockchain) != strings.ToLower(chain) {
			continue
		}
		matches = append(matches, i)
	}

	if len(matches) == 0 {
		if chain != "" {
			return nil, &ResolveError{Code: "TOKEN_NOT_FOUND", Message: fmt.Sprintf("no token %q on chain %q", identifier, chain)}
		}
		return nil, &ResolveError{Code: "TOKEN_NOT_FOUND", Message: fmt.Sprintf("no token %q found", identifier)}
	}
	if len(matches) > 1 {
		chains := make([]string, len(matches))
		for i, idx := range matches {
			chains[i] = tokens[idx].Blockchain
		}
		return nil, &ResolveError{
			Code:    "TOKEN_AMBIGUOUS",
			Message: fmt.Sprintf("%q matches %d tokens on chains: %s — use --from-chain/--to-chain to disambiguate", identifier, len(matches), strings.Join(chains, ", ")),
		}
	}

	return &tokens[matches[0]], nil
}

// convertAmount converts a human-readable amount (e.g., "1.5") to smallest
// units string given the token's decimal places.
func convertAmount(amount string, decimals int) (string, error) {
	// Split on decimal point
	parts := strings.SplitN(amount, ".", 2)
	whole := parts[0]
	if whole == "" {
		whole = "0"
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}

	// Pad or truncate fractional part to exactly `decimals` digits
	if len(frac) > decimals {
		frac = frac[:decimals]
	}
	for len(frac) < decimals {
		frac += "0"
	}

	// Combine and parse as big.Int (removes leading zeros)
	raw := whole + frac
	result := new(big.Int)
	_, ok := result.SetString(raw, 10)
	if !ok {
		return "", fmt.Errorf("invalid amount %q", amount)
	}
	return result.String(), nil
}
