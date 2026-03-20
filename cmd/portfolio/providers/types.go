package providers

// ChainBalance represents balances for a single chain.
type ChainBalance struct {
	Chain    string         `json:"chain"`
	Address  string         `json:"address"`
	TotalUsd float64        `json:"totalUsd"`
	Tokens   []TokenBalance `json:"tokens"`
}

// TokenBalance represents a single token balance.
type TokenBalance struct {
	Symbol          string  `json:"symbol"`
	Balance         string  `json:"balance"`
	Usd             float64 `json:"usd"`
	ContractAddress string  `json:"contractAddress,omitempty"`
	AssetId         string  `json:"assetId,omitempty"`
}

// ProviderError represents a per-chain error in the response.
type ProviderError struct {
	Chain   string `json:"chain"`
	Message string `json:"message"`
}
