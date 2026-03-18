# near-intents-cli

A standalone CLI for cross-chain token swaps via the [NEAR Intents](https://near-intents.org/) (Defuse Protocol 1Click) API. Designed as an agent-first tool — every command returns structured JSON — but works for humans too.

## Install

### From source

```bash
git clone https://github.com/FlipsideCrypto/near-intents-cli.git
cd near-intents-cli
make build
# optionally symlink to PATH
make dev-link
```

### From release

Download the binary for your platform from [Releases](https://github.com/FlipsideCrypto/near-intents-cli/releases).

## Authentication

Get a JWT token from the [Partner Dashboard](https://partners.near-intents.org/). Set it via:

```bash
# Environment variable (recommended for CI)
export NEAR_INTENTS_JWT_TOKEN=<jwt>

# Config file
echo '{"token": "<jwt>"}' > ~/.near-intents.json

# Flag (per-command)
near-intents --token <jwt> tokens
```

Without a token, everything still works — swaps just incur a platform fee.

## Quick start

```bash
# Search for tokens
near-intents tokens --search USDC --chain eth

# Get a quote (dry run, no commitment)
near-intents quote --from USDC --from-chain eth --to wNEAR --to-chain near --amount 10

# Execute the swap
near-intents swap --from USDC --from-chain eth --to wNEAR --to-chain near --amount 10 \
  --recipient alice.near --refund-to 0xYourEthAddress

# User signs at the signingUrl from the response, then:
near-intents submit-tx --deposit-address <addr> --tx-hash <hash>

# Poll for completion
near-intents status --deposit-address <addr>
```

## Commands

| Command | Description |
|---------|-------------|
| `tokens` | List/search supported tokens |
| `quote` | Get a swap quote (dry run) |
| `swap` | Execute a swap (generates deposit address + signing URL) |
| `submit-tx` | Submit deposit tx hash (optional, speeds up processing) |
| `status` | Check swap progress |
| `llm onboard` | Print agent onboarding documentation |

Run `near-intents <command> --help` for full flag details.

## Output format

Every command returns a JSON envelope:

```json
{"success": true, "data": {...}, "error": null}
```

On failure:

```json
{"success": false, "data": null, "error": {"code": "QUOTE_FAILED", "message": "..."}}
```

Use `--pretty` for indented output.

## Token resolution

Tokens can be specified two ways:

- **By asset ID**: `--from nep141:wrap.near`
- **By symbol + chain**: `--from USDC --from-chain eth`

The API uses short chain names: `eth`, `arb`, `base`, `sol`, `near`, `pol`, `bsc`, `op`, `gnosis`, `aptos`.

Use `near-intents tokens` to discover available tokens and their asset IDs.

## Agent integration

Run `near-intents llm onboard` to get structured onboarding context for LLM agents. This covers the full swap workflow, all commands, token resolution, signing flow, status states, and error recovery.

## Config

`~/.near-intents.json`:

```json
{
  "token": "eyJ...",
  "api_endpoint": "https://1click.chaindefuser.com",
  "signing_base_url": "https://swap.flipsidecrypto.xyz"
}
```

All fields are optional. Defaults point to the production API.

## Development

```bash
make build       # Build binary
make test        # Run tests
make ci          # fmt + vet + test + build
make dev-link    # Symlink binary to /usr/local/bin
```

Cross-platform releases are configured via [GoReleaser](.goreleaser.yml).
