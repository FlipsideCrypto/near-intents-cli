package providers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const intentsContract = "intents.near"

// QueryIntentsBalances fetches deposit balances from the intents.near contract for a given account.
func QueryIntentsBalances(account string, tokenIDs []string, tokenRegistry map[string]NEARTokenInfo) (*ChainBalance, error) {
	cb := &ChainBalance{
		Chain:   "intents",
		Address: account,
	}

	if len(tokenIDs) == 0 {
		return cb, nil
	}

	// Encode args as base64 JSON.
	argsJSON, err := json.Marshal(map[string]interface{}{
		"account_id": account,
		"token_ids":  tokenIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal mt_batch_balance_of args: %w", err)
	}
	argsB64 := base64.StdEncoding.EncodeToString(argsJSON)

	// Call mt_batch_balance_of on intents.near.
	result, err := doNearRPC("query", map[string]interface{}{
		"request_type": "call_function",
		"finality":     "final",
		"account_id":   intentsContract,
		"method_name":  "mt_batch_balance_of",
		"args_base64":  argsB64,
	})
	if err != nil {
		return nil, fmt.Errorf("mt_batch_balance_of rpc: %w", err)
	}

	// The RPC result wraps the return value as {"result": [...bytes...], ...}.
	var rpcResult struct {
		Result []byte `json:"result"`
	}
	if err := json.Unmarshal(result, &rpcResult); err != nil {
		return nil, fmt.Errorf("parse rpc result wrapper: %w", err)
	}

	// The bytes are a JSON array of balance strings.
	var balances []string
	if err := json.Unmarshal(rpcResult.Result, &balances); err != nil {
		return nil, fmt.Errorf("parse balance array: %w", err)
	}

	for i, tokenID := range tokenIDs {
		if i >= len(balances) {
			break
		}
		raw := balances[i]
		if raw == "" || raw == "0" {
			continue
		}

		info, ok := tokenRegistry[tokenID]
		if !ok {
			continue
		}

		balStr := FormatBalance(raw, info.Decimals)
		balF := ParseFloat(balStr)
		if balF == 0 {
			continue
		}

		usd := balF * float64(info.Price)

		tb := TokenBalance{
			Symbol:  info.Symbol,
			Balance: balStr,
			Usd:     usd,
			AssetId: info.AssetId,
		}
		cb.Tokens = append(cb.Tokens, tb)
		cb.TotalUsd += usd
	}

	return cb, nil
}
