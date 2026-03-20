# Portfolio Tool — Agent Onboarding

You have access to `portfolio`, a CLI for querying token balances across multiple blockchains.

## The Agentic Loop

The portfolio tool is the OBSERVE step in the trading workflow:

```
OBSERVE  → portfolio balances              (what does the user hold?)
DECIDE   → near-intents intel              (how should they rebalance?)
PLAN     → near-intents quote              (what will swaps cost?)
CONFIRM  → present plan to user            (your responsibility)
EXECUTE  → near-intents swap + submit-tx   (do it)
VERIFY   → near-intents status             (confirm completion)
```

**Never skip OBSERVE.** Every portfolio conversation starts with `portfolio balances`.

## Setup

Before querying balances, the user must configure their wallet addresses:

### Check existing config
```
portfolio setup --list
```

If empty, walk the user through adding their addresses:

### Add addresses
```
portfolio setup --add --chain near --address alice.near
portfolio setup --add --chain evm --address 0xabc...
portfolio setup --add --chain solana --address 7xKX...
portfolio setup --add --chain bitcoin --address bc1q...
```

Chain types:
- `near` — NEAR Protocol account (e.g., `alice.near`, `bob.tg`)
- `evm` — Ethereum/Base/Polygon/Arbitrum/Optimism (one address covers all EVM chains)
- `solana` — Solana wallet address
- `bitcoin` — Bitcoin address

### API Keys Required

For EVM and Solana balance queries, the user needs free API keys:
- **Ankr** (EVM): Set `ANKR_API_KEY` env var or add `ankr_api_key` to `~/.portfolio.json`
- **Helius** (Solana): Set `HELIUS_API_KEY` env var or add `helius_api_key` to `~/.portfolio.json`

NEAR and Bitcoin queries require no API keys.

## Querying Balances

```
portfolio balances                    # all chains
portfolio balances --chain near       # NEAR wallet + intents
portfolio balances --chain evm        # all EVM chains
portfolio balances --chain solana     # Solana
portfolio balances --chain bitcoin    # Bitcoin
portfolio balances --intents-only     # just intents.near deposits
```

The output includes:
- `totalUsd` — total portfolio value
- Per-chain breakdown with individual token balances
- `assetId` fields on NEAR tokens (feed directly into `near-intents swap --from`)
- `near-intents` as a separate "chain" showing tokens deposited in the intents trading engine

### Partial Failures

If a provider fails (e.g., Ankr is down), the command returns whatever data it could get plus an `errors` array. An agent should note the error but still reason about available data.

## Feeding Balances Into Intel

After getting balances, summarize the portfolio and ask Flipside for recommendations:

```
near-intents intel --message "Here's my portfolio:
- 1.5 ETH ($4,800) on Ethereum
- 500 NEAR ($2,500) in NEAR wallet
- 200 wNEAR ($1,000) in intents balance
- 0.004 BTC ($400)
Total: $8,700

How should I rebalance for moderate risk?"
```

The intel command returns analytical recommendations. Use `near-intents quote` to price out each recommended swap before presenting to the user.

## Important Boundaries

- **Portfolio reads, near-intents writes.** This tool queries balances. Swaps happen through `near-intents`.
- **Always confirm with the user** before executing any swaps based on recommendations.
- **intents vs wallet:** Tokens in `near-intents` chain are immediately swappable via native mode. Tokens in `near` chain need deposit steps first.

## Commands Reference

### `portfolio setup`
| Flag | Description |
|------|-------------|
| `--add` | Add an address |
| `--remove` | Remove an address |
| `--list` | List configured addresses |
| `--chain` | Chain type: near, evm, solana, bitcoin |
| `--address` | Wallet address |

### `portfolio balances`
| Flag | Description |
|------|-------------|
| `--chain` | Filter: near, evm, solana, bitcoin |
| `--intents-only` | Show only intents.near balances |

## Output Format

Every command returns JSON:
```json
{"success": true, "data": {...}}
```
or on failure:
```json
{"success": false, "error": {"code": "ERROR_CODE", "message": "details"}}
```

Use `--pretty` for indented output. Default is compact JSON.
