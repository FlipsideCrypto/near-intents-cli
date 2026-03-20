package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/FlipsideCrypto/near-intents-cli/cmd/portfolio/providers"
	"github.com/spf13/cobra"
)

const OneClickAPIEndpoint = "https://1click.chaindefuser.com"

var (
	flagBalancesChain   string
	flagBalancesIntents bool
)

var balancesCmd = &cobra.Command{
	Use:   "balances",
	Short: "Show multi-chain token balances",
	Long: `Query token balances across all configured chains. Fans out requests
to multiple providers in parallel.

Examples:
  portfolio balances
  portfolio balances --chain near
  portfolio balances --chain evm
  portfolio balances --intents-only`,
	RunE: runBalances,
}

func init() {
	balancesCmd.Flags().StringVar(&flagBalancesChain, "chain", "", "Filter by chain: near, evm, solana, bitcoin")
	balancesCmd.Flags().BoolVar(&flagBalancesIntents, "intents-only", false, "Show only intents.near balances")
	rootCmd.AddCommand(balancesCmd)
}

type balancesOutput struct {
	TotalUsd float64                   `json:"totalUsd"`
	Errors   []providers.ProviderError `json:"errors,omitempty"`
	Chains   []providers.ChainBalance  `json:"chains"`
}

type oneClickToken struct {
	AssetId         string  `json:"assetId"`
	Decimals        int     `json:"decimals"`
	Blockchain      string  `json:"blockchain"`
	Symbol          string  `json:"symbol"`
	Price           float32 `json:"price"`
	ContractAddress *string `json:"contractAddress,omitempty"`
}

func runBalances(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()

	if len(cfg.Addresses) == 0 && cfg.NearIntentsAccount == "" {
		printError("NO_ADDRESSES", "No addresses configured. Run: portfolio setup --add --chain <chain> --address <addr>")
		return nil
	}

	// nearRegistry is keyed by bare contract address (for QueryNEARBalances).
	// intentsRegistry is keyed by "nep141:<contract>" (for QueryIntentsBalances).
	nearRegistry, intentsRegistry, nearPrice, nearTokenIDs := fetchTokenRegistry()

	var (
		mu     sync.Mutex
		chains []providers.ChainBalance
		errors []providers.ProviderError
		wg     sync.WaitGroup
	)

	addChain := func(cb *providers.ChainBalance) {
		mu.Lock()
		defer mu.Unlock()
		if cb != nil && len(cb.Tokens) > 0 {
			chains = append(chains, *cb)
		}
	}
	addChains := func(cbs []providers.ChainBalance) {
		mu.Lock()
		defer mu.Unlock()
		for _, cb := range cbs {
			if len(cb.Tokens) > 0 {
				chains = append(chains, cb)
			}
		}
	}
	addError := func(chain, msg string) {
		mu.Lock()
		defer mu.Unlock()
		errors = append(errors, providers.ProviderError{Chain: chain, Message: msg})
	}

	shouldQuery := func(chain string) bool {
		if flagBalancesIntents {
			return chain == "near-intents"
		}
		if flagBalancesChain == "" {
			return true
		}
		return flagBalancesChain == chain
	}

	findAddr := func(chain string) string {
		for _, a := range cfg.Addresses {
			if a.Chain == chain {
				return a.Address
			}
		}
		return ""
	}

	nearAccount := findAddr("near")
	intentsAccount := cfg.NearIntentsAccount
	if intentsAccount == "" {
		intentsAccount = nearAccount
	}

	// EVM
	if shouldQuery("evm") {
		if addr := findAddr("evm"); addr != "" {
			ankrKey := resolveAnkrAPIKey(cfg)
			if ankrKey == "" {
				addError("evm", "ANKR_API_KEY not configured")
			} else {
				wg.Add(1)
				go func() {
					defer wg.Done()
					verbose("querying Ankr for EVM balances: %s", addr)
					cbs, err := providers.QueryEVMBalances(addr, ankrKey)
					if err != nil {
						addError("evm", err.Error())
						return
					}
					addChains(cbs)
				}()
			}
		}
	}

	// Solana
	if shouldQuery("solana") {
		if addr := findAddr("solana"); addr != "" {
			heliusKey := resolveHeliusAPIKey(cfg)
			if heliusKey == "" {
				addError("solana", "HELIUS_API_KEY not configured")
			} else {
				wg.Add(1)
				go func() {
					defer wg.Done()
					verbose("querying Helius for Solana balances: %s", addr)
					cb, err := providers.QuerySolanaBalances(addr, heliusKey)
					if err != nil {
						addError("solana", err.Error())
						return
					}
					addChain(cb)
				}()
			}
		}
	}

	// Bitcoin
	if shouldQuery("bitcoin") {
		if addr := findAddr("bitcoin"); addr != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				verbose("querying mempool.space for BTC balance: %s", addr)
				cb, err := providers.QueryBitcoinBalance(addr)
				if err != nil {
					addError("bitcoin", err.Error())
					return
				}
				addChain(cb)
			}()
		}
	}

	// NEAR wallet
	if shouldQuery("near") && nearAccount != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			verbose("querying NEAR balances: %s", nearAccount)
			cb, err := providers.QueryNEARBalances(nearAccount, nearRegistry, nearPrice)
			if err != nil {
				addError("near", err.Error())
				return
			}
			addChain(cb)
		}()
	}

	// NEAR Intents
	if (shouldQuery("near") || flagBalancesIntents) && intentsAccount != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			verbose("querying intents.near balances: %s", intentsAccount)
			cb, err := providers.QueryIntentsBalances(intentsAccount, nearTokenIDs, intentsRegistry)
			if err != nil {
				addError("near-intents", err.Error())
				return
			}
			addChain(cb)
		}()
	}

	wg.Wait()

	var totalUsd float64
	for _, cb := range chains {
		totalUsd += cb.TotalUsd
	}

	if len(chains) == 0 && len(errors) > 0 {
		printError("ALL_PROVIDERS_FAILED", fmt.Sprintf("%d provider(s) failed", len(errors)))
		return nil
	}

	out := balancesOutput{
		TotalUsd: totalUsd,
		Errors:   errors,
		Chains:   chains,
	}
	printSuccess(out)
	return nil
}

// fetchTokenRegistry fetches token metadata from the 1Click API and returns:
//   - nearRegistry: keyed by bare contract address (for QueryNEARBalances)
//   - intentsRegistry: keyed by "nep141:<contract>" (for QueryIntentsBalances)
//   - nearPrice: USD price of native NEAR
//   - nearTokenIDs: slice of "nep141:<contract>" token IDs for intents batch query
func fetchTokenRegistry() (map[string]providers.NEARTokenInfo, map[string]providers.NEARTokenInfo, float32, []string) {
	nearRegistry := make(map[string]providers.NEARTokenInfo)
	intentsRegistry := make(map[string]providers.NEARTokenInfo)
	var nearPrice float32
	var nearTokenIDs []string

	url := OneClickAPIEndpoint + "/v0/tokens"
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		verbose("failed to fetch token registry: %v", err)
		return nearRegistry, intentsRegistry, nearPrice, nearTokenIDs
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nearRegistry, intentsRegistry, nearPrice, nearTokenIDs
	}

	var tokens []oneClickToken
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nearRegistry, intentsRegistry, nearPrice, nearTokenIDs
	}

	for _, t := range tokens {
		if t.Blockchain == "near" {
			if t.Symbol == "NEAR" {
				nearPrice = t.Price
			}
			if t.ContractAddress != nil {
				info := providers.NEARTokenInfo{
					Symbol:   t.Symbol,
					Decimals: t.Decimals,
					Price:    t.Price,
					AssetId:  t.AssetId,
				}
				nearRegistry[*t.ContractAddress] = info
				nep141Key := "nep141:" + *t.ContractAddress
				intentsRegistry[nep141Key] = info
				nearTokenIDs = append(nearTokenIDs, nep141Key)
			}
		}
	}

	return nearRegistry, intentsRegistry, nearPrice, nearTokenIDs
}
