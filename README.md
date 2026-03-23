# near-intents-cli

A standalone CLI for cross-chain token swaps via [NEAR Intents](https://near-intents.org/) (Defuse Protocol 1Click API), plus a multi-chain portfolio balance viewer. Designed agent-first — every command returns structured JSON — but works for humans too.

## Install

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh
```

This installs both `near-intents` and `portfolio` binaries to `/usr/local/bin`.

Custom install dir (no sudo needed):
```bash
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh
```

Specific version:
```bash
VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh
```

### From source

Requires Go 1.24+:

```bash
git clone https://github.com/FlipsideCrypto/near-intents-cli.git
cd near-intents-cli
make build build-portfolio
```

### From release

Download binaries for your platform from [Releases](https://github.com/FlipsideCrypto/near-intents-cli/releases).

## Two tools

| Tool | Purpose |
|------|---------|
| `near-intents` | Cross-chain token swaps, quotes, status tracking, portfolio intel |
| `portfolio` | Multi-chain balance queries (NEAR, EVM, Solana, Bitcoin) |

## Quick start

```bash
# Search for tokens
near-intents tokens --search USDC --chain eth

# Check NEAR balances (wallet + intents)
near-intents balances --account alice.near

# Get a quote (dry run)
near-intents quote --from USDC --from-chain eth --to wNEAR --to-chain near --amount 10

# Execute the swap
near-intents swap --from USDC --from-chain eth --to wNEAR --to-chain near --amount 10 \
  --recipient alice.near --refund-to 0xYourEthAddress

# User signs at the signingUrl from the response, then:
near-intents submit-tx --deposit-address <addr> --tx-hash <hash>

# Poll for completion
near-intents status --deposit-address <addr>
```

### Portfolio setup

```bash
# Add your wallet addresses
portfolio setup --add --chain near --address alice.near
portfolio setup --add --chain evm --address 0xabc...
portfolio setup --add --chain solana --address 7xKX...
portfolio setup --add --chain bitcoin --address bc1q...

# View all balances across chains
portfolio balances --pretty
```

## Agent integration (skill)

This repo includes an [agent skill](https://agentskills.io) at `skills/near-intents-trading/` that teaches LLM agents how to orchestrate both tools.

Install in Claude Code:
```bash
claude skills add github.com/FlipsideCrypto/near-intents-cli/skills/near-intents-trading
```

The skill follows the loop:

```
OBSERVE  → portfolio balances / near-intents balances
DECIDE   → near-intents intel (Flipside AI recommendations)
PLAN     → near-intents quote (price out each swap)
CONFIRM  → present plan to user with fees
EXECUTE  → near-intents swap + submit-tx
VERIFY   → near-intents status
```

Each tool also has built-in onboarding docs for agents:

```bash
near-intents llm onboard    # swap CLI reference + gotchas
portfolio llm onboard       # portfolio CLI reference
near-intents llm topics     # deep-dive topics (native swaps, near-cli)
```

## Authentication

Get a JWT from the [Partner Dashboard](https://partners.near-intents.org/):

```bash
export NEAR_INTENTS_JWT_TOKEN=<jwt>
# or: echo '{"token": "<jwt>"}' > ~/.near-intents.json
# or: near-intents --jwt <token> <command>
```

Without a token, everything works — swaps just incur a platform fee.

## Commands — near-intents

| Command | Description |
|---------|-------------|
| `tokens` | List/search supported tokens |
| `balances` | Show NEAR wallet + intents balances for an account |
| `quote` | Get a swap quote (dry run) |
| `swap` | Execute a swap (deposit address + signing URL) |
| `submit-tx` | Submit deposit tx hash (speeds up processing) |
| `status` | Check swap progress |
| `intel` | Get portfolio recommendations from Flipside AI |
| `llm onboard` | Print agent onboarding documentation |
| `llm topics` | List deep-dive documentation topics |
| `llm topic <name>` | Print a specific topic |

## Commands — portfolio

| Command | Description |
|---------|-------------|
| `setup` | Add/remove/list wallet addresses |
| `balances` | Query balances across all configured chains |
| `llm onboard` | Print agent onboarding documentation |

### Portfolio API keys

EVM and Solana balance queries need free API keys:

```bash
export ANKR_API_KEY=<key>      # get at ankr.com — EVM balances
export HELIUS_API_KEY=<key>    # get at helius.dev — Solana balances
```

Or add to `~/.portfolio.json`. NEAR and Bitcoin queries need no keys.

## Output format

Every command returns a JSON envelope:

```json
{"success": true, "data": {...}}
```

On failure:

```json
{"success": false, "error": {"code": "QUOTE_FAILED", "message": "..."}}
```

Use `--pretty` for indented output.

## Token resolution

Tokens can be specified two ways:

- **By asset ID**: `--from nep141:wrap.near`
- **By symbol + chain**: `--from USDC --from-chain eth`

Chain names: `eth`, `arb`, `base`, `sol`, `near`, `pol`, `bsc`, `op`, `gnosis`, `aptos`.

Use `near-intents tokens` to discover available tokens and their asset IDs.

## Config files

**`~/.near-intents.json`** — near-intents config:
```json
{
  "token": "eyJ...",
  "flipside_api_key": "...",
  "api_endpoint": "https://1click.chaindefuser.com",
  "signing_base_url": "https://swap.flipsidecrypto.xyz"
}
```

**`~/.portfolio.json`** — portfolio config:
```json
{
  "addresses": [
    {"chain": "near", "address": "alice.near"},
    {"chain": "evm", "address": "0xabc..."}
  ],
  "ankr_api_key": "",
  "helius_api_key": "",
  "near_intents_account": "alice.near"
}
```

All fields are optional.

## Development

```bash
make build             # Build near-intents
make build-portfolio   # Build portfolio
make test              # Run tests
make ci                # fmt + vet + test + build + build-portfolio
make dev-link          # Symlink near-intents to /usr/local/bin
make dev-link-portfolio # Symlink portfolio to /usr/local/bin
```

Cross-platform releases via [GoReleaser](.goreleaser.yml) — builds both binaries for linux/darwin × amd64/arm64.
